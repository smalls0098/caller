.PHONY: models
models:
	cd apipb && protoc --go_out=. models.proto

.PHONY: build
build:
	mkdir -p bin/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o ./bin/caller ./cmd