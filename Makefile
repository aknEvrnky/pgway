.PHONY: proto build test tools

build:
	go build -o build/pgway ./cmd/pgway
	go build -o build/pgway-cp ./cmd/pgway-cp
	go build -o build/pgway-dp ./cmd/pgway-dp
	go build -o build/pgctl ./cmd/pgctl

# Dev tools (GOBIN / $(go env GOPATH)/bin must be on PATH).
# protoc itself is a system package (not installable via go install) —
# see the check at the end of this target.
# Plugin pins match current go.mod / generated-code toolchain (bump when upgrading).
tools:
	go install gotest.tools/gotestsum@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.2
	@command -v protoc >/dev/null 2>&1 || { \
		echo 'error: protoc not found on PATH'; \
		echo 'Install the protobuf compiler, then re-run make tools:'; \
		echo '  macOS:  brew install protobuf'; \
		echo '  Debian: sudo apt-get install -y protobuf-compiler'; \
		echo '  docs:   https://grpc.io/docs/protoc-installation/'; \
		exit 1; \
	}

test: tools
	gotestsum -- -race ./...

proto: tools
	protoc \
		--proto_path=proto \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		proto/pgway/controlplane/v1/*.proto
