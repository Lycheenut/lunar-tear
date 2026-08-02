# Proto generation: outputs to gen/proto/ (module=lunar-tear/server).
# All proto files have go_package. Only protos used by the server are generated here
# (generating all would put them in one package and cause name clashes).
PROTO_USED = proto/banner.proto proto/battle.proto proto/bighunt.proto proto/cageornament.proto proto/character.proto proto/characterboard.proto proto/characterviewer.proto proto/companion.proto proto/config.proto proto/consumableitem.proto proto/contentsstory.proto proto/costume.proto proto/data.proto proto/deck.proto proto/dokan.proto proto/explore.proto proto/friend.proto proto/gacha.proto proto/gameplay.proto proto/gift.proto proto/gimmick.proto proto/labyrinth.proto proto/loginbonus.proto proto/material.proto proto/mission.proto proto/movie.proto proto/navicutin.proto proto/omikuji.proto proto/notification.proto proto/parts.proto proto/portalcage.proto proto/pvp.proto proto/quest.proto proto/reward.proto proto/shop.proto proto/sidestoryquest.proto proto/tutorial.proto proto/user.proto proto/weapon.proto

PROTOC ?= protoc
GOOSE ?= goose
GO ?= go

# Android client patching. Paths are relative to server/.
ifeq ($(OS),Windows_NT)
  PYTHON ?= python
else
  PYTHON ?= python3
endif
APKTOOL ?= apktool
ZIPALIGN ?= zipalign
APKSIGNER ?= apksigner
KEYTOOL ?= keytool

INPUT_APK ?= ../client/client.apk
OUTPUT_APK ?= ../client/client-patched.apk
PATCHED_UNALIGNED_APK ?= $(basename $(OUTPUT_APK))-unaligned$(suffix $(OUTPUT_APK))
PATCHED_DIR ?= $(basename $(OUTPUT_APK))-work
PATCH_APK_SCRIPT ?= ../scripts/android/patch_apk.py
KEYSTORE ?= ../client/debug.keystore
KEY_ALIAS ?= key
KEYSTORE_PASS ?= android
KEY_PASS ?= android

# Master data patching. Paths are relative to server/.
PATCH_MASTERDATA_SCRIPT ?= ../scripts/patch_masterdata.py
MASTERDATA_INSTALLER ?= hack/install_masterdata.py
MASTERDATA_INPUT ?= assets/release/20240404193219.bin.e
MASTERDATA_TARGET ?= assets/release/20240404193219.bin.e
MASTERDATA_RELOAD_URL ?= http://127.0.0.1:8082/api/admin/master-data/reload

# Background lifecycle for the existing three-service dev supervisor.
DEV_MANAGER ?= hack/manage_dev.py
DEV_RUN_DIR ?= tmp/run
DEV_PID_FILE ?= $(DEV_RUN_DIR)/dev.pid
DEV_READY_FILE ?= $(DEV_RUN_DIR)/dev.ready
DEV_STOP_FILE ?= $(DEV_RUN_DIR)/dev.stop
DEV_ARGS_FILE ?= $(DEV_RUN_DIR)/dev.args.json
DEV_LOG_FILE ?= $(DEV_RUN_DIR)/dev.log
DEV_START_TIMEOUT ?= 90
DEV_STOP_TIMEOUT ?= 30

# Production lifecycle via Docker Compose. Paths are relative to server/.
DOCKER_COMPOSE ?= docker compose
GCLOUD ?= gcloud
PROD_ENV_FILE ?= deploy/.env.production
PROD_COMPOSE_FILE ?= deploy/docker-compose.production.yaml
PROD_COMPOSE = $(DOCKER_COMPOSE) --env-file "$(PROD_ENV_FILE)" -f "$(PROD_COMPOSE_FILE)"
PROD_ADMIN_TOKEN_SECRET ?= LUNAR_ADMIN_TOKEN
PROD_ADMIN_TOKEN_VERSION ?= 1
GCP_PROJECT_ID ?=
PROD_GCP_PROJECT_ARG = $(if $(strip $(GCP_PROJECT_ID)),--project "$(GCP_PROJECT_ID)")
PROD_ADMIN_TOKEN_COMMAND = $(GCLOUD) secrets versions access "$(PROD_ADMIN_TOKEN_VERSION)" --secret="$(PROD_ADMIN_TOKEN_SECRET)" $(PROD_GCP_PROJECT_ARG)

GRPC_ADDR ?= 10.0.2.2:8003
GRPC_TLS ?= false
HTTP_ADDR ?= 10.0.2.2:8080
AUTH_HOST ?= 10.0.2.2:3000
DEFAULT_TEXT_LANGUAGE ?= en
DEFAULT_VOICE_LANGUAGE ?= en
PATCH_AUTH_ARG = $(if $(strip $(AUTH_HOST)),--auth-host "$(AUTH_HOST)")
PATCH_GRPC_TLS_ARG = $(if $(filter 1 true TRUE yes YES,$(GRPC_TLS)),--grpc-tls)
PATCH_LANGUAGE_ARGS = --default-text-language "$(DEFAULT_TEXT_LANGUAGE)" --default-voice-language "$(DEFAULT_VOICE_LANGUAGE)"

# Android client patching. Paths are relative to server/.
ifeq ($(OS),Windows_NT)
  PYTHON ?= python
else
  PYTHON ?= python3
endif
APKTOOL ?= apktool
ZIPALIGN ?= zipalign
APKSIGNER ?= apksigner
KEYTOOL ?= keytool

INPUT_APK ?= ../client/client.apk
OUTPUT_APK ?= ../client/client-patched.apk
PATCHED_UNALIGNED_APK ?= $(basename $(OUTPUT_APK))-unaligned$(suffix $(OUTPUT_APK))
PATCHED_DIR ?= $(basename $(OUTPUT_APK))-work
PATCH_APK_SCRIPT ?= ../scripts/android/patch_apk.py
KEYSTORE ?= ../client/debug.keystore
KEY_ALIAS ?= key
KEYSTORE_PASS ?= android
KEY_PASS ?= android

GRPC_ADDR ?= 10.0.2.2:8003
GRPC_TLS ?= false
HTTP_ADDR ?= 10.0.2.2:8080
AUTH_HOST ?= 10.0.2.2:3000
DEFAULT_TEXT_LANGUAGE ?= en
DEFAULT_VOICE_LANGUAGE ?= en
PATCH_AUTH_ARG = $(if $(strip $(AUTH_HOST)),--auth-host "$(AUTH_HOST)")
PATCH_GRPC_TLS_ARG = $(if $(filter 1 true TRUE yes YES,$(GRPC_TLS)),--grpc-tls)
PATCH_LANGUAGE_ARGS = --default-text-language "$(DEFAULT_TEXT_LANGUAGE)" --default-voice-language "$(DEFAULT_VOICE_LANGUAGE)"

EXE =
ifeq ($(OS),Windows_NT)
  EXE = .exe
endif

proto:
	$(PROTOC) -I . $(PROTO_USED) --go_out=. --go_opt=module=lunar-tear/server --go-grpc_out=. --go-grpc_opt=module=lunar-tear/server
	@echo "Generated in gen/proto/"

build:
	go build -o lunar-tear$(EXE) ./cmd/lunar-tear

build-cdn:
	go build -o octo-cdn$(EXE) ./cmd/octo-cdn

build-import:
	go build -o import-snapshot$(EXE) ./cmd/import-snapshot

build-auth:
	go build -o auth-server$(EXE) ./cmd/auth-server

build-claim-account:
	go build -o claim-account$(EXE) ./cmd/claim-account

build-register-account:
	go build -o register-account$(EXE) ./cmd/register-account

build-dev:
	$(GO) build -o bin/dev$(EXE) ./cmd/dev

build-all:
	go build -o bin/dev$(EXE) ./cmd/dev
	go build -o bin/auth-server$(EXE) ./cmd/auth-server
	go build -o bin/octo-cdn$(EXE) ./cmd/octo-cdn
	go build -o bin/lunar-tear$(EXE) ./cmd/lunar-tear

ifeq ($(OS),Windows_NT)
client-check:
	@if not exist "$(INPUT_APK)" (echo Missing input APK: $(INPUT_APK) 1>&2 & exit /b 1)
	@if not exist "$(PATCH_APK_SCRIPT)" (echo Missing patch script: $(PATCH_APK_SCRIPT) 1>&2 & exit /b 1)
	@where "$(firstword $(PYTHON))" >nul 2>&1 || (echo Missing Python command: $(PYTHON) 1>&2 & exit /b 1)
	@where "$(firstword $(APKTOOL))" >nul 2>&1 || (echo Missing apktool command: $(APKTOOL) 1>&2 & exit /b 1)
	@where "$(firstword $(ZIPALIGN))" >nul 2>&1 || (echo Missing zipalign command: $(ZIPALIGN) 1>&2 & exit /b 1)
	@where "$(firstword $(APKSIGNER))" >nul 2>&1 || (echo Missing apksigner command: $(APKSIGNER) 1>&2 & exit /b 1)
	@if not exist "$(KEYSTORE)" (where "$(firstword $(KEYTOOL))" >nul 2>&1 || (echo Missing keytool command: $(KEYTOOL) 1>&2 & exit /b 1))
	@if "$(GRPC_ADDR)"=="" (echo GRPC_ADDR must not be empty 1>&2 & exit /b 1)
	@if "$(HTTP_ADDR)"=="" (echo HTTP_ADDR must not be empty 1>&2 & exit /b 1)
else
client-check:
	@test -f "$(INPUT_APK)" || { echo "Missing input APK: $(INPUT_APK)" >&2; exit 1; }
	@test -f "$(PATCH_APK_SCRIPT)" || { echo "Missing patch script: $(PATCH_APK_SCRIPT)" >&2; exit 1; }
	@command -v "$(PYTHON)" >/dev/null 2>&1 || { echo "Missing Python command: $(PYTHON)" >&2; exit 1; }
	@command -v "$(APKTOOL)" >/dev/null 2>&1 || { echo "Missing apktool command: $(APKTOOL)" >&2; exit 1; }
	@command -v "$(ZIPALIGN)" >/dev/null 2>&1 || { echo "Missing zipalign command: $(ZIPALIGN)" >&2; exit 1; }
	@command -v "$(APKSIGNER)" >/dev/null 2>&1 || { echo "Missing apksigner command: $(APKSIGNER)" >&2; exit 1; }
	@if [ ! -f "$(KEYSTORE)" ]; then \
		command -v "$(KEYTOOL)" >/dev/null 2>&1 || { echo "Missing keytool command: $(KEYTOOL)" >&2; exit 1; }; \
	fi
	@test -n "$(GRPC_ADDR)" || { echo "GRPC_ADDR must not be empty" >&2; exit 1; }
	@test -n "$(HTTP_ADDR)" || { echo "HTTP_ADDR must not be empty" >&2; exit 1; }
endif

# Usage:
#   make client GRPC_ADDR=10.0.2.2:8003 GRPC_TLS=false HTTP_ADDR=10.0.2.2:8080 AUTH_HOST=10.0.2.2:3000 DEFAULT_TEXT_LANGUAGE=ja DEFAULT_VOICE_LANGUAGE=ja INPUT_APK=/path/to/original.apk OUTPUT_APK=/path/to/patched.apk
# Set GRPC_TLS=true for a TLS reverse proxy and AUTH_HOST= to skip the Facebook login redirect patch.
ifeq ($(OS),Windows_NT)
client: client-check
	@if not exist "$(dir $(OUTPUT_APK))" mkdir "$(dir $(OUTPUT_APK))"
	@if not exist "$(dir $(KEYSTORE))" mkdir "$(dir $(KEYSTORE))"
	$(APKTOOL) d "$(INPUT_APK)" -o "$(PATCHED_DIR)" -f
	$(PYTHON) "$(PATCH_APK_SCRIPT)" "$(PATCHED_DIR)" --grpc-addr "$(GRPC_ADDR)" $(PATCH_GRPC_TLS_ARG) --http-addr "$(HTTP_ADDR)" $(PATCH_AUTH_ARG) $(PATCH_LANGUAGE_ARGS)
	$(APKTOOL) b "$(PATCHED_DIR)" -o "$(PATCHED_UNALIGNED_APK)"
	$(ZIPALIGN) -p -f 4 "$(PATCHED_UNALIGNED_APK)" "$(OUTPUT_APK)"
	@if not exist "$(KEYSTORE)" (echo Generating signing keystore: $(KEYSTORE) & $(KEYTOOL) -genkeypair -noprompt -keystore "$(KEYSTORE)" -alias "$(KEY_ALIAS)" -storepass "$(KEYSTORE_PASS)" -keypass "$(KEY_PASS)" -keyalg RSA -keysize 2048 -validity 10000 -dname "CN=Debug")
	$(APKSIGNER) sign --ks "$(KEYSTORE)" --ks-key-alias "$(KEY_ALIAS)" --ks-pass pass:"$(KEYSTORE_PASS)" --key-pass pass:"$(KEY_PASS)" "$(OUTPUT_APK)"
	$(ZIPALIGN) -c -p 4 "$(OUTPUT_APK)"
	$(APKSIGNER) verify "$(OUTPUT_APK)"
	@echo Patched APK: $(OUTPUT_APK)
else
client: client-check
	@mkdir -p "$(dir $(OUTPUT_APK))" "$(dir $(KEYSTORE))"
	$(APKTOOL) d "$(INPUT_APK)" -o "$(PATCHED_DIR)" -f
	$(PYTHON) "$(PATCH_APK_SCRIPT)" "$(PATCHED_DIR)" --grpc-addr "$(GRPC_ADDR)" $(PATCH_GRPC_TLS_ARG) --http-addr "$(HTTP_ADDR)" $(PATCH_AUTH_ARG) $(PATCH_LANGUAGE_ARGS)
	$(APKTOOL) b "$(PATCHED_DIR)" -o "$(PATCHED_UNALIGNED_APK)"
	$(ZIPALIGN) -p -f 4 "$(PATCHED_UNALIGNED_APK)" "$(OUTPUT_APK)"
	@if [ ! -f "$(KEYSTORE)" ]; then \
		echo "Generating signing keystore: $(KEYSTORE)"; \
		$(KEYTOOL) -genkeypair -noprompt -keystore "$(KEYSTORE)" \
			-alias "$(KEY_ALIAS)" -storepass "$(KEYSTORE_PASS)" -keypass "$(KEY_PASS)" \
			-keyalg RSA -keysize 2048 -validity 10000 -dname "CN=Debug"; \
	fi
	$(APKSIGNER) sign --ks "$(KEYSTORE)" --ks-key-alias "$(KEY_ALIAS)" \
		--ks-pass pass:"$(KEYSTORE_PASS)" --key-pass pass:"$(KEY_PASS)" "$(OUTPUT_APK)"
	$(ZIPALIGN) -c -p 4 "$(OUTPUT_APK)"
	$(APKSIGNER) verify "$(OUTPUT_APK)"
	@echo "Patched APK: $(OUTPUT_APK)"
endif

# Patch to a temporary file, validate it, atomically install it at the path
# consumed by both lunar-tear and octo-cdn, then hot-reload when
# LUNAR_ADMIN_TOKEN is present. Set MASTERDATA_RELOAD_URL= to disable reload.
# Usage:
#   make masterdata MASTERDATA_INPUT=/path/to/original.bin.e
masterdata:
	$(PYTHON) "$(MASTERDATA_INSTALLER)" --patcher "$(PATCH_MASTERDATA_SCRIPT)" --input "$(MASTERDATA_INPUT)" --target "$(MASTERDATA_TARGET)" --reload-url="$(MASTERDATA_RELOAD_URL)"

# Run cmd/dev as the background supervisor for auth-server, octo-cdn, and
# lunar-tear. ARGS is forwarded unchanged to cmd/dev.
start:
	$(PYTHON) "$(DEV_MANAGER)" --go "$(GO)" --binary "bin/dev$(EXE)" --pid-file "$(DEV_PID_FILE)" --ready-file "$(DEV_READY_FILE)" --stop-file "$(DEV_STOP_FILE)" --args-file "$(DEV_ARGS_FILE)" --log-file "$(DEV_LOG_FILE)" --start-timeout "$(DEV_START_TIMEOUT)" --stop-timeout "$(DEV_STOP_TIMEOUT)" start -- $(ARGS)

stop:
	$(PYTHON) "$(DEV_MANAGER)" --pid-file "$(DEV_PID_FILE)" --ready-file "$(DEV_READY_FILE)" --stop-file "$(DEV_STOP_FILE)" --args-file "$(DEV_ARGS_FILE)" --log-file "$(DEV_LOG_FILE)" --stop-timeout "$(DEV_STOP_TIMEOUT)" stop

restart:
	$(PYTHON) "$(DEV_MANAGER)" --go "$(GO)" --binary "bin/dev$(EXE)" --pid-file "$(DEV_PID_FILE)" --ready-file "$(DEV_READY_FILE)" --stop-file "$(DEV_STOP_FILE)" --args-file "$(DEV_ARGS_FILE)" --log-file "$(DEV_LOG_FILE)" --start-timeout "$(DEV_START_TIMEOUT)" --stop-timeout "$(DEV_STOP_TIMEOUT)" restart -- $(ARGS)

prod-build:
	@LUNAR_ADMIN_TOKEN=unused-during-build $(PROD_COMPOSE) build

prod-start:
	@LUNAR_ADMIN_TOKEN="$$($(PROD_ADMIN_TOKEN_COMMAND))" || exit $$?; \
		test -n "$$LUNAR_ADMIN_TOKEN" || { echo "Secret $(PROD_ADMIN_TOKEN_SECRET) version $(PROD_ADMIN_TOKEN_VERSION) is empty" >&2; exit 1; }; \
		export LUNAR_ADMIN_TOKEN; \
		$(PROD_COMPOSE) up -d --no-build

prod-stop:
	@LUNAR_ADMIN_TOKEN=unused-during-stop $(PROD_COMPOSE) stop

prod-restart:
	@LUNAR_ADMIN_TOKEN="$$($(PROD_ADMIN_TOKEN_COMMAND))" || exit $$?; \
		test -n "$$LUNAR_ADMIN_TOKEN" || { echo "Secret $(PROD_ADMIN_TOKEN_SECRET) version $(PROD_ADMIN_TOKEN_VERSION) is empty" >&2; exit 1; }; \
		export LUNAR_ADMIN_TOKEN; \
		$(PROD_COMPOSE) up -d --no-build --force-recreate

prod-deploy: prod-build
	@$(MAKE) --no-print-directory prod-start

clean:
	rm -rf bin/*

dev:
	go run ./cmd/dev $(ARGS)

migrate:
	$(GOOSE) -dir migrations -allow-missing sqlite3 db/game.db up

restore:
	go run ./cmd/wizard-restore

import:
ifndef SNAPSHOT
	$(error SNAPSHOT is required, e.g. make import SNAPSHOT=snapshots/scene_1.json UUID=...)
endif
ifndef UUID
	$(error UUID is required, e.g. make import SNAPSHOT=snapshots/scene_1.json UUID=...)
endif
	go run ./cmd/import-snapshot --snapshot $(SNAPSHOT) --uuid $(UUID)

.PHONY: proto build build-cdn build-auth build-import build-claim-account build-register-account build-dev build-all client-check client masterdata start stop restart prod-build prod-start prod-stop prod-restart prod-deploy clean dev migrate restore import
