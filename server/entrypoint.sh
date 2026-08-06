#!/usr/bin/env sh
set -e

mkdir -p db
goose -dir migrations sqlite3 db/game.db up

AUTH_FLAG=""
if [ -n "${LUNAR_AUTH_URL}" ]; then
  AUTH_FLAG="--auth-url ${LUNAR_AUTH_URL}"
fi

ADMIN_FLAG=""
if [ -n "${LUNAR_ADMIN_LISTEN}" ]; then
  ADMIN_FLAG="--admin-listen ${LUNAR_ADMIN_LISTEN}"
fi

GACHA_CONFIG_FLAG=""
if [ -n "${LUNAR_GACHA_CONFIG}" ]; then
  GACHA_CONFIG_FLAG="--gacha-config ${LUNAR_GACHA_CONFIG}"
fi

REGISTER_FLAG=""
case "${LUNAR_NO_REGISTER:-}" in
  1|true|TRUE|yes|YES)
    REGISTER_FLAG="--no-register"
    ;;
esac

exec ./lunar-tear \
  --listen "${LUNAR_LISTEN:-0.0.0.0:443}" \
  --public-addr "${LUNAR_PUBLIC_ADDR}" \
  --octo-url "${LUNAR_OCTO_URL}" \
  ${AUTH_FLAG} \
  ${ADMIN_FLAG} \
  ${GACHA_CONFIG_FLAG} \
  ${REGISTER_FLAG}
