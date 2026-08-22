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

QUEST_DROP_CONFIG_FLAG=""
if [ -n "${LUNAR_QUEST_DROP_CONFIG}" ]; then
  QUEST_DROP_CONFIG_FLAG="--quest-drop-config ${LUNAR_QUEST_DROP_CONFIG}"
fi

exec ./lunar-tear \
  --listen "${LUNAR_LISTEN:-0.0.0.0:443}" \
  --public-addr "${LUNAR_PUBLIC_ADDR}" \
  --octo-url "${LUNAR_OCTO_URL}" \
  ${AUTH_FLAG} \
  ${ADMIN_FLAG} \
  ${QUEST_DROP_CONFIG_FLAG}
