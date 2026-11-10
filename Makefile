.ONESHELL:

export PODMAN_COMPOSE_WARNING_LOGS = false

COMPOSE = podman compose -f .container/compose.yaml --env-file .container/env

up:
	@$(COMPOSE) up --build --detach

down:
	@$(COMPOSE) down

shell:
	@$(COMPOSE) exec dev sh

build:
	@$(COMPOSE) exec -T dev sh -ec "
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o bin/darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o bin/darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/windows-amd64.exe main.go
	"
