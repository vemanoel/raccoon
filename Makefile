.ONESHELL:

export PODMAN_COMPOSE_WARNING_LOGS=false

up:
	@podman compose up --build --detach

down:
	@podman compose down

shell:
	@podman compose exec dev sh

build:
	@podman compose exec -T development sh -ec "
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o bin/darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o bin/darwin-arm64 main.go
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/linux-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/windows-amd64.exe main.go
	"
