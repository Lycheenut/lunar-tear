# Production deployment files

These files keep the gRPC, auth, CDN, and admin ports on the ECS loopback
interface. Nginx is the only public application listener.

## 1. Prepare the host

Create the persistent directories and make them writable by container UID 1000:

```sh
sudo install -d -o 1000 -g 1000 /srv/lunar-tear/db
sudo install -d -o 1000 -g 1000 /srv/lunar-tear/assets
```

Copy the required `assets/` tree to `/srv/lunar-tear/assets/`.

## 2. Configure the deployment

```sh
cd server/deploy
cp .env.production.example .env.production
chmod 600 .env.production
```

Replace every example domain and token in `.env.production` and
`nginx/lunar-tear.conf`. Set `AUTH_ALLOWED_REDIRECT_URIS` to the exact Android
and iOS callback URIs emitted by the patched clients. Install a Cloudflare
Origin CA certificate covering the three ECS hostnames.

## 3. Validate and start

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

## 4. Prepare R2 objects

The R2 custom domain must be short enough for the fixed-width URL in
`list.bin`. The command pads a shorter URL with an `r` path segment and creates
matching object keys:

```sh
cd server
go run ./cmd/prepare-r2 \
  --assets-dir . \
  --resources-base-url https://assets.example.com \
  --resource-version 300116832 \
  --dry-run
```

After validation succeeds, omit `--dry-run` and select an empty output
directory:

```sh
go run ./cmd/prepare-r2 \
  --assets-dir . \
  --resources-base-url https://assets.example.com \
  --resource-version 300116832 \
  --output r2-publish
```

Upload the generated directory to the bucket root. The command fails before
writing files if an object is missing, its MD5 is invalid, or Android and iOS
map different bytes to the same public URL. In that collision case, keep
serving assets through `octo-cdn` until a platform-aware Worker/router exists.

## 5. Backups

Back up `game.db`, `auth.db`, `auth.secret`, and the master-data file. Use an
SQLite online backup or stop the containers before copying the database files.
Test restoration on a separate instance.
