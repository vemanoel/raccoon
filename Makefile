.ONESHELL:
.SILENT:

export

PODMAN_COMPOSE_PROVIDER ?= $(shell command -v podman-compose)
PODMAN_COMPOSE_WARNING_LOGS = false
COMPOSE_FILE ?= .container/compose.dev.yaml

COMPOSE_CMD = podman compose

ifeq ($(COMPOSE_FILE),.container/compose.dev.yaml)
    COMPOSE_CMD += --env-file .container/env
endif

up:
	$(COMPOSE_CMD) up --build --detach

down:
	$(COMPOSE_CMD) down

shell:
	$(COMPOSE_CMD) exec dev sh

build:
	$(COMPOSE_CMD) exec -T dev sh -ec "
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o bin/darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags '-s -w' -o bin/darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o bin/linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o bin/windows-amd64.exe main.go
	"
