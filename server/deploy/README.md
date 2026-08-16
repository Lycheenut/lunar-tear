# Production deployment files

These files keep the gRPC, auth, CDN, and admin ports on the ECS loopback
interface. Nginx is the only public application listener.

Resource bundles must be uploaded from the local workstation directly to R2.
They are never copied to or served by ECS. ECS only needs the master-data file
and `list.bin`.

## 1. Prepare and upload R2 objects locally

Create the R2 bucket and attach its production custom domain before running
this command. The custom domain must be short enough for the fixed-width URL
in `list.bin`. The command pads a shorter URL with an `r` path segment and
creates matching object keys:

The bundled client requests `unso-200116832-*`, so publish with Octo asset
version `200116832`. This compatibility value is independent of the
master-data version.

```sh
cd server
go run ./cmd/prepare-r2 \
  --assets-dir . \
  --resources-base-url https://assets.example.com \
  --resource-version 200116832 \
  --dry-run
```

After validation succeeds, omit `--dry-run` and select an empty local output
directory:

```sh
go run ./cmd/prepare-r2 \
  --assets-dir . \
  --resources-base-url https://assets.example.com \
  --resource-version 200116832 \
  --output tmp/r2-publish
```

The command validates four files concurrently by default and prints progress,
elapsed time, and ETA for both validation and materialization. Use
`--workers N` to tune the concurrency for the local disk.

Some R2 object IDs differ only by letter case. On Windows, the command checks
the empty output directory before hashing and automatically tries to enable
NTFS per-directory case sensitivity. If that requires elevation, run the
reported `fsutil.exe file setCaseSensitiveInfo ... enable` command once from an
Administrator PowerShell, then rerun `prepare-r2`.

Configure an R2 remote named `r2`, then upload from Windows PowerShell on the
local workstation directly to the `lunar-tear` bucket. Do not route this
transfer through ECS:

```powershell
rclone copy .\tmp\r2-publish r2:lunar-tear `
  --exclude manifest.json `
  --transfers 32 `
  --checkers 64 `
  --fast-list `
  --progress `
  --header-upload "Cache-Control: public, max-age=31536000, immutable" `
  --header-upload "Content-Type: application/octet-stream"

rclone check .\tmp\r2-publish r2:lunar-tear `
  --exclude manifest.json `
  --one-way `
  --checksum
```

The preparation command fails before writing files if an object is missing,
its MD5 is invalid, or Android and iOS map different bytes to the same public
URL. In that collision case, keep serving assets through `octo-cdn` until a
platform-aware Worker/router exists.

Disable the `r2.dev` development URL after testing. In Cloudflare, add a Cache
Rule for the R2 custom hostname that marks the resource path as eligible for
cache; extensionless object keys are not reliably cached by the default file
extension rules.

## 2. Prepare the host

Create the persistent directories and make them writable by container UID
1000:

```sh
sudo install -d -o 1000 -g 1000 /srv/lunar-tear/db
sudo install -d -o 1000 -g 1000 /srv/lunar-tear/assets/release
sudo install -d -o 1000 -g 1000 /srv/lunar-tear/assets/revisions/0
```

Copy only these runtime files to ECS, preserving their relative paths:

```text
assets/release/20240404193219.bin.e
assets/revisions/0/list.bin
```

If the local asset set has platform-specific lists, copy
`assets/revisions/0/android/list.bin` and
`assets/revisions/0/ios/list.bin` instead. Do not copy the `assetbundle/` or
`resources/` directories to ECS.

## 3. Configure the deployment

```sh
cd server/deploy
cp .env.production.example .env.production
chmod 600 .env.production
```

Replace every example domain in `.env.production` and
`nginx/lunar-tear.conf`. `LUNAR_ADMIN_TOKEN` is read from Google Cloud Secret
Manager at deployment time and must not be added to `.env.production`. The
Android WebView login flow uses
`fbconnect://success`, which is already set in the example. If another client
uses a different login path, set `AUTH_ALLOWED_REDIRECT_URIS` to its exact
`redirect_uri` value as well. Install a Cloudflare Origin CA certificate
covering the four ECS hostnames: `grpc.example.com`, `auth.example.com`,
`octo.example.com`, and `admin.example.com`. The nginx template exposes the
management UI only through `https://admin.example.com/admin/`; port 8082
remains bound to loopback.

Set `OCTO_RESOURCES_BASE_URL` to the R2 custom domain used in step 1, not the
ECS `octo` hostname.

### Configure Cloud Logging

The production Compose template sends container stdout and stderr directly to
Google Cloud Logging with Docker's `gcplogs` driver. Set `GCP_PROJECT_ID` in
`.env.production` to the same project passed to the production Makefile
targets. The Docker host identity needs `roles/logging.logWriter` in that
project.

On Compute Engine, grant that role to the VM's attached service account. On a
host outside Google Cloud, configure Application Default Credentials for the
Docker daemon itself; setting credentials only inside the application
containers does not authenticate the logging driver. Protect any credential
configuration as a root-readable secret and restart Docker after changing the
daemon environment.

The logging driver includes the `environment` and `service` container labels.
It uses a 4 MB non-blocking buffer per container so a temporary logging outage
does not block application stdout or stderr. When that buffer fills, Docker
drops new log messages, so monitor ingestion and alert on sustained failures.

### Query production logs through the local MCP server

The repository includes a read-only STDIO MCP server. It uses local Application
Default Credentials and calls Cloud Logging directly; it does not expose a
network port and does not run in the production Compose stack.

Build it locally from the `server` directory:

```sh
make build-log-mcp
```

If GNU Make is not installed, use PowerShell:

```powershell
New-Item -ItemType Directory -Force .\bin | Out-Null
go build -o .\bin\log-mcp.exe .\cmd\log-mcp
```

The credential used on the workstation needs `logging.logEntries.list`, which
is included in `roles/logging.viewer`. Keep the project fixed in the MCP
process configuration rather than accepting it as a tool argument. Add a
project-level `.codex/config.toml` using absolute paths:

```toml
[mcp_servers.production_logs]
command = '/absolute/path/to/lunar-tear/server/bin/log-mcp'
args = ["--project", "your-google-cloud-project"]
startup_timeout_sec = 20
tool_timeout_sec = 45
```

On Windows, point `command` to the absolute `log-mcp.exe` path. Restart Codex
after adding the configuration, then verify that these tools are available:

- `list_services` lists the fixed production service allowlist.
- `search_logs` searches one service, defaults to 15 minutes, and limits a
  query to one hour and 200 entries.
- `get_log_context` returns entries around an RFC3339 timestamp.

Before starting Codex, verify the exact ADC identity used by the MCP process:

```sh
gcloud auth application-default print-access-token
```

This is separate from the credential used by a normal `gcloud logging read`
command. If ADC impersonates a service account, the calling identity also
needs `roles/iam.serviceAccountTokenCreator` on that service account. Do not
put an access token in `.codex/config.toml`; let the Google client refresh ADC.

Docker's `gcplogs` driver stores configured container labels inside the JSON
payload. The MCP server therefore filters on
`jsonPayload.container.metadata.environment` and
`jsonPayload.container.metadata.service`, not `labels.environment` or
`labels.service`. Returned messages are size-limited and common token-like
values are redacted before they reach the MCP client.

## 4. Validate and start

```sh
cd ..
make prod-deploy \
  GCP_PROJECT_ID=your-google-cloud-project \
  PROD_ADMIN_TOKEN_VERSION=1
```

The VM service account must have `roles/secretmanager.secretAccessor` on the
`LUNAR_ADMIN_TOKEN` secret. `prod-build` never reads the secret or passes it to
the image build. `prod-start` reads it immediately before creating containers,
and `prod-restart` reads it again before recreating them. Set
`PROD_ADMIN_TOKEN_SECRET` only when the Secret Manager secret uses another ID.
The same `GCP_PROJECT_ID` must be present in `.env.production` so Compose can
configure `gcplogs`.

After the containers start, verify the driver before querying Logs Explorer:

```sh
docker inspect --format '{{json .HostConfig.LogConfig}}' \
  "$(docker compose --env-file deploy/.env.production \
    -f deploy/docker-compose.production.yaml ps -q server)"
```

The output should report `gcplogs`, the configured GCP project, and the
`environment,service` label list. Because logging-driver changes require
container recreation, use `make prod-restart` when applying this change to an
existing deployment.

The auth token secret is generated once at `${DATA_DIR}/db/auth.secret` with
mode 0600 and reused after restarts. Do not delete it while users have active
tokens.

Install `nginx/lunar-tear.conf`, validate with `nginx -t`, and reload nginx.
Restrict ECS port 443 to Cloudflare IP ranges and port 22 to administrator IPs.
Do not open ports 3000, 8003, 8080, or 8082.

The public admin hostname still requires `LUNAR_ADMIN_TOKEN` for every data API
call. Use a high-entropy generated token and put a Cloudflare Access
self-hosted application in front of `admin.example.com` as a second
authentication layer. Do not create a Cache Rule for this hostname; the admin
responses are marked `no-store`.

Build the Android client with `--grpc-tls` when its gRPC address points to this
Cloudflare/nginx TLS endpoint. The patcher's default plaintext gRPC mode is for
direct local-server connections and will be rejected before nginx can log an
HTTP/2 request. Build from a fresh original APK through the repository's
canonical target:

```sh
cd server
make client \
  INPUT_APK=../client/client.apk \
  OUTPUT_APK=../client/client-production.apk \
  GRPC_ADDR=grpc.example.com:443 \
  GRPC_TLS=true \
  HTTP_ADDR=octo.example.com:443 \
  AUTH_HOST=auth.example.com:443
```

The target requires Python 3, `apktool`, `zipalign`, `apksigner`, and `keytool`.
Use `DEFAULT_TEXT_LANGUAGE` and `DEFAULT_VOICE_LANGUAGE` to change the initial
language, or override the keystore variables in `server/Makefile` for signing.

Add a Cloudflare Cache Rule for the `octo` hostname and `/v1/list/` and
`/v2/.../list/` request paths so repeat downloads are served from the edge.
Keep its edge TTL short enough to purge or roll out a changed list safely.

## 5. Configure Cloudflare DNS for the admin hostname

In the Cloudflare dashboard, open the zone and add this record under
**DNS > Records**:

| Type | Name | Content | Proxy status | TTL |
| ---- | ---- | ------- | ------------ | --- |
| `A` | `admin` | the ECS public IPv4 address | Proxied | Auto |

Add a proxied `AAAA` record as well only when the ECS host has a working public
IPv6 address. If an existing origin hostname already resolves to the same host,
a proxied `CNAME` from `admin` to that hostname is also valid. Never switch this
record to DNS only when the origin uses a Cloudflare Origin CA certificate.

Then:

1. Under **SSL/TLS > Overview**, select **Full (strict)**.
2. Confirm the installed origin certificate includes `admin.example.com` (or a
   matching `*.example.com` SAN), then reload nginx.
3. Open `https://admin.example.com/admin/` and enter the token sourced from
   Secret Manager. A request without the token should receive HTTP 401 from the
   data API.
4. Under **Zero Trust > Access controls > Applications**, create a
   **Self-hosted** public-hostname application for `admin.example.com`, and add
   an Allow policy containing only the administrator identities. Access is an
   additional gate; it does not replace `LUNAR_ADMIN_TOKEN`.
5. Keep the origin firewall restricted to current Cloudflare IP ranges on port
   443. Ports 8082 and the other application ports must remain closed publicly.

## 6. Backups

Back up `game.db`, `auth.db`, `auth.secret`, and the master-data file. Use an
SQLite online backup or stop the containers before copying the database files.
Test restoration on a separate instance.
