.PHONY: proto build test

build:
	go build -o build/pgway ./cmd/pgway
	go build -o build/pgway-cp ./cmd/pgway-cp
	go build -o build/pgway-dp ./cmd/pgway-dp
	go build -o build/pgctl ./cmd/pgctl

test:
	go test -race ./...

proto:
	protoc \
		--proto_path=proto \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		proto/pgway/controlplane/v1/*.proto
