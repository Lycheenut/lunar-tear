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

Replace every example domain and token in `.env.production` and
`nginx/lunar-tear.conf`. The Android WebView login flow uses
`fbconnect://success`, which is already set in the example. If another client
uses a different login path, set `AUTH_ALLOWED_REDIRECT_URIS` to its exact
`redirect_uri` value as well. Install a Cloudflare Origin CA certificate
covering the three ECS hostnames.

Set `OCTO_RESOURCES_BASE_URL` to the R2 custom domain used in step 1, not the
ECS `octo` hostname.

## 4. Validate and start

```sh
docker compose --env-file .env.production \
  -f docker-compose.production.yaml config
docker compose --env-file .env.production \
  -f docker-compose.production.yaml up -d --build
```

The auth token secret is generated once at `${DATA_DIR}/db/auth.secret` with
mode 0600 and reused after restarts. Do not delete it while users have active
tokens.

Install `nginx/lunar-tear.conf`, validate with `nginx -t`, and reload nginx.
Restrict ECS port 443 to Cloudflare IP ranges and port 22 to administrator IPs.
Do not open ports 3000, 8003, 8080, or 8082.

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

## 5. Backups

Back up `game.db`, `auth.db`, `auth.secret`, and the master-data file. Use an
SQLite online backup or stop the containers before copying the database files.
Test restoration on a separate instance.
