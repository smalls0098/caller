.PHONY: models
models:
	cd apipb && protoc --go_out=. models.proto
