# Lunar Tear

Private server research project for a certain discontinued mobile game.
Discord server: https://discord.gg/MZAf5aVkJG

## How To Launch The Server

### Download & Run (no setup)

Prebuilt binaries are published for Linux, macOS, and Windows on the [Releases page](https://github.com/Walter-Sparrow/lunar-tear/releases).

1. Download the archive for your OS/arch (`lunar-tear-server-<version>-<os>-<arch>.{tar.gz,zip}`).
2. Run `./wizard` (macOS/Linux) or double-click `wizard.exe` (Windows).

### Prerequisites (build from source)

- Go 1.25+
- [goose](https://github.com/pressly/goose) migration tool
- Populated `server/assets/` directory

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

### Quick Start (Wizard)

The interactive wizard walks you through setup with a few simple questions — no flags or networking knowledge needed. It auto-detects the right IP address for your emulator or phone and launches all services.

```bash
cd server
go run ./cmd/wizard
```

Your choices are saved so next time you just press Enter to relaunch with the same settings. To skip the confirmation prompt entirely (useful for scripts or quick relaunches), pass `--prefer-saved`:

```bash
go run ./cmd/wizard --prefer-saved
```

If no saved config exists, the flag prints an error and exits.

#### Custom Ports

By default the wizard uses ports 8003 (gRPC), 8080 (CDN), and 3000 (auth). Override any of them with flags:

```bash
go run ./cmd/wizard --grpc-port 9003 --cdn-port 9080
```

| Flag             | Default | Description                                                                                                              |
| ---------------- | ------- | ------------------------------------------------------------------------------------------------------------------------ |
| `--prefer-saved` | `false` | Reuse saved config without prompting                                                                                     |
| `--grpc-port`    | `8003`  | gRPC server port                                                                                                         |
| `--cdn-port`     | `8080`  | CDN server port                                                                                                          |
| `--auth-port`    | `3000`  | Auth server port                                                                                                         |
| `--admin-port`   | `0`     | Admin management port (`0` = disabled). Bound on `127.0.0.1`; only takes effect when `LUNAR_ADMIN_TOKEN` is set in the env. |

Custom ports are saved to `.wizard.json` alongside your other settings. On the next run the saved ports are reused automatically — no need to pass the flags again. If you later pass different port flags, the wizard warns you that the ports changed and asks for confirmation before continuing.

### Regenerate protobuf stubs

```bash
cd server
make proto
```

### Database

Player state is stored in a SQLite database. Run migrations before starting the server:

```bash
cd server
make migrate
```

Or manually:

```bash
cd server
mkdir -p db
goose -dir migrations -allow-missing sqlite3 db/game.db up
```

### Backups & Restore

The wizard backs up your save every time you launch it. To roll back to an earlier save:

```bash
cd server
make restore
```

Pick a backup from the list and confirm.

### Importing a Snapshot

To import a JSON snapshot into the database, use the import tool. The `--uuid` flag must match the UUID your game client sends during authentication:

```bash
cd server
make import SNAPSHOT=snapshots/scene_1.json UUID=<your-client-uuid>
```

Or directly:

```bash
go run ./cmd/import-snapshot \
  --snapshot snapshots/scene_1.json \
  --uuid <your-client-uuid> \
  --db db/game.db
```

| Flag         | Default      | Description                                   |
| ------------ | ------------ | --------------------------------------------- |
| `--snapshot` | _(required)_ | Path to JSON snapshot file                    |
| `--uuid`     | _(required)_ | UUID to assign (must match the client's UUID) |
| `--db`       | `db/game.db` | SQLite database path                          |

### Run

The server is split into two binaries: a gRPC game server and an HTTP asset CDN. Both must be running for the client to work.

**Start the CDN** (serves asset bundles, list.bin, master data, web pages):

```bash
cd server
go run ./cmd/octo-cdn \
  --listen 0.0.0.0:8080 \
  --public-addr 10.0.2.2:8080
```

**Start the game server** (gRPC, points the client at the CDN):

```bash
cd server
go run ./cmd/lunar-tear \
  --listen 0.0.0.0:8003 \
  --public-addr 10.0.2.2:8003 \
  --octo-url http://10.0.2.2:8080
```

The default listen address is `0.0.0.0:443`, which requires `sudo` (privileged port). Use `--listen` with a high port to avoid this. If you do need port 443, either use `sudo` or grant the binary the capability on Linux:

```bash
go build -o lunar-tear ./cmd/lunar-tear
sudo setcap cap_net_bind_service=+ep ./lunar-tear
./lunar-tear --public-addr 10.0.2.2:443 --octo-url http://10.0.2.2:8080
```

The CDN can run on a completely separate machine — just set `--octo-url` on the game server and `--public-addr` on the CDN to the externally-reachable address.

### Run All Services At Once

Instead of starting each service individually, use the dev runner to launch all three (auth, CDN, game server) with a single command. No Docker required — works on macOS, Linux, and Windows.

```bash
cd server
make dev
```

Or directly:

```bash
cd server
go run ./cmd/dev
```

Each service's output is prefixed with a colored label (`[auth]`, `[cdn]`, `[grpc]`). Press Ctrl+C to shut everything down.

The dev runner automatically builds each service into `bin/` before launching. This means the binaries have stable file paths, so **Windows Firewall only prompts once** — subsequent runs reuse the same allowed executables. The wizard performs the same build step transparently.

Override defaults with namespaced flags:

```bash
go run ./cmd/dev --grpc.listen 0.0.0.0:9000 --grpc.public-addr 10.0.2.2:9000 --cdn.public-addr 192.168.1.50:8080
```

Or via `make`:

```bash
make dev ARGS="--grpc.listen 0.0.0.0:9000 --grpc.public-addr 10.0.2.2:9000"
```

For background operation, `make start` detects the host's active LAN IPv4 and
uses it for missing `--cdn.public-addr` and `--grpc.public-addr` values. Explicit
values in `ARGS` take precedence. `make restart` re-detects the address.

| Flag                 | Default                 | Description                                                                                                          |
| -------------------- | ----------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `--auth.listen`      | `0.0.0.0:3000`          | auth-server listen address                                                                                           |
| `--auth.db`          | `db/auth.db`            | auth-server SQLite database path                                                                                     |
| `--cdn.listen`       | `0.0.0.0:8080`          | octo-cdn local bind address                                                                                          |
| `--cdn.public-addr`  | `10.0.2.2:8080`         | octo-cdn externally-reachable addr                                                                                   |
| `--grpc.listen`      | `0.0.0.0:8003`          | lunar-tear gRPC listen address                                                                                       |
| `--grpc.public-addr` | `10.0.2.2:8003`         | lunar-tear externally-reachable addr                                                                                 |
| `--grpc.octo-url`    | `http://10.0.2.2:8080`  | Octo CDN base URL passed to lunar-tear                                                                               |
| `--grpc.auth-url`    | `http://localhost:3000` | auth server base URL passed to lunar-tear                                                                            |
| `--no-register`      | `false`                 | disable new user registrations (only already registered users can connect).                                          |
| `--admin.listen`     | _(empty)_               | lunar-tear admin HTTP bind. Empty = leave default; the listener only binds when `LUNAR_ADMIN_TOKEN` is set in the env. |
| `--no-color`         | `false`                 | disable colored output                                                                                               |

### Ports

| Protocol | Port | Binary        | Notes                                                                                                            |
| -------- | ---- | ------------- | ---------------------------------------------------------------------------------------------------------------- |
| gRPC     | 443  | `lunar-tear`  | default; configurable with `--listen` (requires patched client)                                                  |
| HTTP     | 8080 | `octo-cdn`    | Octo asset API + game web pages                                                                                  |
| HTTP     | 8082 | `lunar-tear`  | master-data management UI (`/admin/`) and API; loopback by default, token-gated and opt-in |
| HTTP     | 3000 | `auth-server` | account registration and login                                                                                   |

### Game Server Flags (`lunar-tear`)

| Flag             | Default          | Description                                                                 |
| ---------------- | ---------------- | --------------------------------------------------------------------------- |
| `--listen`       | `0.0.0.0:443`    | gRPC listen address (host:port)                                             |
| `--public-addr`  | `127.0.0.1:443`  | externally-reachable host:port advertised to clients                        |
| `--octo-url`     | _(required)_     | CDN base URL the client uses for assets (e.g. `http://10.0.2.2:8080`)       |
| `--db`           | `db/game.db`     | SQLite database path                                                        |
| `--auth-url`     | _(empty)_        | Auth server base URL (e.g. `http://localhost:3000`)                         |
| `--admin-listen` | `127.0.0.1:8082` | Admin UI/API listen address. Only binds when `LUNAR_ADMIN_TOKEN` is set.    |
| `--gacha-config` | `config/gacha.json` | Plaintext Gacha classification, limited-set, and pickup configuration.    |
| `--no-register`  | `false`          | Disable new user registrations (only already registered users can connect). |

### Master Data Management and Live Reload

The game server reads its master data from `assets/release/20240404193219.bin.e` at startup. When `LUNAR_ADMIN_TOKEN` is set, open [http://127.0.0.1:8082/admin/](http://127.0.0.1:8082/admin/) and enter that token to manage limited-time content.

The management UI reads every table whose schema has a matching `*StartDatetime` / `*EndDatetime` pair and exposes all datetime fields on those rows. Where a row references localized text assets, the UI also shows its English title by default and can switch to Japanese or Korean. Shop and shop-item-cell-term rows include their `shop → cell group → cell` relationships. Saving performs the following as one guarded operation:

1. Validates the submitted fields and start/end ordering (an end value of `0` keeps its existing disabled-row meaning).
2. Rebuilds and encrypts a candidate `.bin.e` while preserving MessagePack int64 cell widths.
3. Fully loads every runtime catalog from the candidate.
4. Atomically replaces the file shared by the game server and CDN, then publishes the new in-memory catalogs.

The UI uses a content hash for optimistic locking, so a stale browser cannot overwrite a newer master-data file. The asset directory and existing `.bin.e` must be writable by the `lunar-tear` process.

The **Gacha Configuration** tab builds a developer-facing weapon reference from master data (ID, localized weapon name, paired Costume name, attribute, weapon type, and star rating), with filters for attribute, weapon type, and whether a weapon grants a character. Every eligible weapon defaults to standard and therefore needs no JSON entry; developers only store per-weapon event/limited overrides, allowed limited sets, and ordered pickup lists. Root weapons automatically excluded by master data are omitted from both the editor and the published file. The page previews both normal-draw and tenth-draw probabilities before publishing. Publishing validates every override and banner pool, writes `--gacha-config` as readable JSON, and atomically swaps the live catalog; it never modifies or imports these choices into master data. With no configuration file, all eligible weapons remain available as standard weapons and every banner has no limited sets or pickups.

At draw time, the server first selects one of the six `grant type × rarity` groups using the configured group weights. If that group has pickups, the group probability is divided 50/50 between all pickups and all non-pickups, with uniform selection inside each side. On every tenth result, each 2-star group's full weight moves to the 3-star group with the same grant type while 4-star weights stay unchanged. Event-only weapons never enter a pool, standard weapons enter every premium banner, and limited weapons enter only banners that allow their limited set.

To swap in an externally edited file **without restarting** the server:

1. Replace `assets/release/20240404193219.bin.e` on disk with your edited copy.
2. POST to the reload API with a Bearer token matching `LUNAR_ADMIN_TOKEN`:

```bash
curl -X POST -H "Authorization: Bearer ${LUNAR_ADMIN_TOKEN}" \
  http://127.0.0.1:8082/api/admin/master-data/reload
```

The server re-reads the file, atomically swaps every in-memory catalog and derived handler, and bumps the file's mtime. The mtime is folded into `GetLatestMasterDataVersion`, so connected clients see a new version string and re-download the file from the CDN on their next poll.

Security defaults are fail-closed:

- `LUNAR_ADMIN_TOKEN` **must** be set in the environment, or the admin listener never binds.
- `--admin-listen` defaults to `127.0.0.1:8082` (loopback only). Bind to `0.0.0.0` only if you intend to expose it.
- Data APIs use constant-time Bearer-token comparison. The UI stores the supplied token in tab-scoped `sessionStorage`, never in a URL or persistent cookie.

### CDN Flags (`octo-cdn`)

| Flag            | Default          | Description                                               |
| --------------- | ---------------- | --------------------------------------------------------- |
| `--listen`      | `0.0.0.0:8080`   | local bind address                                        |
| `--public-addr` | `127.0.0.1:8080` | externally-reachable address (used in list.bin rewriting) |
| `--resources-base-url` | _(derived from `--public-addr`)_ | public HTTP(S) resource URL embedded in `list.bin`; padded to 43 bytes |
| `--assets-dir`  | `.`              | root directory containing the `assets/` tree              |

### Docker

Three services are available via Docker Compose: the game server (`lunar-tear`), the CDN (`octo-cdn`), and the auth server (`auth-server`). Migrations run automatically on game server start.

```bash
cd server
docker compose up -d
```

The `db/` directory is mounted as a volume so `game.db`, `auth.db`, and the
generated `auth.secret` persist across restarts. Published ports bind to
`127.0.0.1` by default; use a TLS reverse proxy or explicitly override the
bindings when remote access is intended. Make sure `assets/` is populated
before starting.

For the hardened ECS configuration, nginx template, and R2 publication
workflow, see [`server/deploy/README.md`](server/deploy/README.md).

Each service has its own image and can be deployed independently:

| Service  | Image                       | Default Port | Notes                            |
| -------- | --------------------------- | ------------ | -------------------------------- |
| `server` | `kretts/lunar-tear:latest`  | 8003, 8082   | gRPC game server + master-data admin UI/API |
| `cdn`    | `kretts/octo-cdn:latest`    | 8080         | HTTP asset CDN                   |
| `auth`   | `kretts/auth-server:latest` | 3000         | Account registration and login   |

The game server is configured via environment variables in the compose file:

| Env var              | Description                                                                           |
| -------------------- | ------------------------------------------------------------------------------------- |
| `LUNAR_LISTEN`       | gRPC bind address                                                                     |
| `LUNAR_PUBLIC_ADDR`  | Client-facing address advertised to the game                                          |
| `LUNAR_OCTO_URL`     | CDN base URL the client uses for assets                                               |
| `LUNAR_AUTH_URL`     | Auth server base URL (optional)                                                       |
| `LUNAR_ADMIN_LISTEN` | Admin UI/API bind address inside the container (compose default: `0.0.0.0:8082`)      |
| `LUNAR_ADMIN_TOKEN`  | Bearer token for admin data APIs. **The admin listener does not bind unless this is set.** |
| `LUNAR_GACHA_CONFIG` | Plaintext Gacha config path (compose uses `db/gacha.json` so it persists with the DB volume). |

Auth is optional — if `LUNAR_AUTH_URL` is unset the game server starts without it. The admin service is published to `127.0.0.1:8082` on the host so it cannot be reached directly; the production nginx template exposes it at `https://admin.example.com/admin/` through Cloudflare while keeping the Bearer-token API gate. Set `LUNAR_ADMIN_TOKEN` (e.g. via a `.env` file) before bringing the stack up.

### Makefile Targets

All targets run from the `server/` directory.

| Target                        | Description                                            |
| ----------------------------- | ------------------------------------------------------ |
| `make proto`                  | Regenerate protobuf stubs                              |
| `make build`                  | Build the game server binary                           |
| `make build-cdn`              | Build the CDN binary                                   |
| `make build-auth`             | Build the auth server binary                           |
| `make build-dev`              | Build the dev runner binary to `bin/`                  |
| `make build-all`              | Build all service binaries to `bin/`                   |
| `make prod-build`             | Build the production Docker Compose images             |
| `make prod-deploy`            | Build images, read the admin token, and start services  |
| `make prod-start`             | Read the admin token and start production services      |
| `make prod-stop`              | Stop the production services                           |
| `make prod-restart`           | Reload the admin token and recreate production services |
| `make client-check`           | Validate Android client build inputs and tools         |
| `make client`                 | Patch, rebuild, align, and sign the Android client     |
| `make build-import`           | Build the import-snapshot tool                         |
| `make build-claim-account`    | Build the claim-account tool                           |
| `make build-register-account` | Build the register-account tool                        |
| `make clean`                  | Remove the `bin/` directory                            |
| `make dev`                    | Run all three services with one command                |
| `make migrate`                | Run goose migrations on `db/game.db`                   |
| `make restore`                | Interactive restore of `db/game.db` from `db/backups/` |
| `make import`                 | Import a snapshot (`SNAPSHOT=... UUID=...` required)   |

`make client` defaults to local emulator addresses and runs `client-check`
before decoding the APK. Set `GRPC_TLS=true` when `GRPC_ADDR` terminates TLS,
set `AUTH_HOST=` to omit the Facebook login redirect patch, and use
`DEFAULT_TEXT_LANGUAGE` / `DEFAULT_VOICE_LANGUAGE` to select initial language
defaults.

The production start, restart, and deploy targets read `LUNAR_ADMIN_TOKEN`
directly from Google Cloud Secret Manager without writing it to disk. They use
secret `LUNAR_ADMIN_TOKEN`, version `1`, and the active `gcloud` project by
default. Override these with `PROD_ADMIN_TOKEN_SECRET`,
`PROD_ADMIN_TOKEN_VERSION`, and `GCP_PROJECT_ID`.

### Client Localization CSV

The Japanese text additions used for client patching are stored in
`client/localization.csv` as UTF-8 with BOM. CSV localization support is built
into `scripts/android/patch_apk.py`; there is no separate localization packer.

When invoking the Android patcher directly, pass the decompiled APK directory,
server addresses, and CSV path explicitly:

```bash
python3 scripts/android/patch_apk.py patched \
  --grpc-addr 10.0.2.2:8003 \
  --http-addr 10.0.2.2:8080 \
  --localization-csv client/localization.csv
```

`apk_dir`, `--grpc-addr`, and `--http-addr` retain the original required
interface. `--localization-csv` also has no default path; omitting it leaves the
localization TextAssets unchanged. Rebuilding, zipalign, and signing continue
to use the existing workflow documented in `scripts/README.md`.

| CSV field | Purpose |
| --------- | ------- |
| `Key` | Client localization key; do not rename it. |
| `Japanese` | Japanese text to add or replace. |

Every row in the CSV is applied to the APK-embedded localization TextAsset
derived from the key prefix. Empty values, unsupported keys, and duplicate keys
are rejected. AssetBundle-only text is not included. Real line breaks in CSV
cells are normalized to the literal `\n` format used by the client catalog.
Re-running the patcher is safe: matching values are reported as
`already-present`; differing values are replaced.

## Claim Account

Transfers an existing game account to the most recently connected client. Looks up a player by their in-game name, assigns the new client's UUID to that account, and deletes the empty account the new client created.

Useful when a new client connects and creates a throwaway account, but you want it to load an existing account instead.

```bash
cd server
go run ./cmd/claim-account --name "PlayerName" --db db/game.db
```

| Flag     | Default      | Description                  |
| -------- | ------------ | ---------------------------- |
| `--name` | _(required)_ | In-game player name to claim |
| `--db`   | `db/game.db` | SQLite database path         |

## Auth Server

A separate HTTP server that handles player account registration and login. The patched client's Facebook login button is redirected to this server, which presents a username/password form. Tokens issued here are validated by the game server to link or recover accounts.

### Run

```bash
cd server
go run ./cmd/auth-server \
  --listen 0.0.0.0:3000 \
  --db db/auth.db
```

By default, the server generates a random token key in `db/auth.secret` and
reuses it across restarts. The key value is never printed. Use `--secret` only
when an external secret manager supplies at least 32 bytes directly.

### Flags

| Flag            | Default        | Description                                                                |
| --------------- | -------------- | -------------------------------------------------------------------------- |
| `--listen`      | `0.0.0.0:3000` | HTTP listen address (host:port)                                            |
| `--db`          | `db/auth.db`   | SQLite database path for auth users                                        |
| `--secret`      | _(empty)_      | Explicit HMAC secret for token signing (minimum 32 bytes)                   |
| `--secret-file` | next to `--db` | Persistent hex-encoded HMAC secret; generated with mode `0600` when absent |
| `--allowed-redirect-uris` | _(compatible Facebook schemes)_ | Comma-separated exact OAuth callbacks; required by the production template |
| `--no-register` | `false`        | Disable new user registrations (only already registered users can log in). |

## Create account

This tool creates a fresh account in main db and new account in Auth Server store with given name & password and automatically binds them together.
A primary mean of registering new accounts when `--no-register` flag is passed to lunar-tear for controlled server access.

```bash
go run ./cmd/register-account --name "AccountName" --password "AccountPassword" --platform "android"
```

| Flag         | Default      | Description                                       |
| ------------ | ------------ | ------------------------------------------------- |
| `--name`     | _(required)_ | Auth Server account nickname to be registered     |
| `--password` | _(required)_ | Auth Server account password to be registered     |
| `--platform` | `android`    | Platform of new user account (`android` or `ios`) |
| `--db`       | `db/game.db` | SQLite main database path                         |
| `--auth-db`  | `db/auth.db` | SQLite Auth Server database path                  |

This only sets the nickname of Auth Server account, a player can choose their in-game nickname upon first login!

## ⚠️ Legal Disclaimer

**Lunar Tear** is a fan-made, non-commercial **preservation and research project** dedicated to keeping a certain discontinued mobile game playable for educational and archival purposes.

- This project is **not affiliated with**, **endorsed by**, or **approved by** the original publisher or any of its subsidiaries.
- All trademarks, copyrights, and intellectual property related to the original game and its associated franchises belong to their respective owners.
- All code in this repository is original work developed through clean-room reverse engineering for interoperability with the game client.
- No copyrighted game assets, binaries, or master data are distributed in this repository.

**Use at your own risk.** The author assumes no liability for any damages or legal consequences that may arise from using this software. By using or contributing to this project, you are solely responsible for ensuring your usage complies with all applicable laws in your jurisdiction.

This project is released under the [MIT License](LICENSE).

**If you are a rights holder with concerns regarding this project**, please contact me directly.
