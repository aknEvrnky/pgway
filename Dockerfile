# syntax=docker/dockerfile:1
# Local / CI image build from source. Release images are built by GoReleaser
# via Dockerfile.goreleaser from prebuilt linux binaries (same ldflags).

FROM golang:1.27-bookworm AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN mkdir -p /out \
	&& LDFLAGS="-s -w -X github.com/aknEvrnky/pgway/internal/platform/version.Version=${VERSION} -X github.com/aknEvrnky/pgway/internal/platform/version.Commit=${COMMIT} -X github.com/aknEvrnky/pgway/internal/platform/version.Date=${DATE}" \
	&& go build -trimpath -ldflags="${LDFLAGS}" -o /out/pgway ./cmd/pgway \
	&& go build -trimpath -ldflags="${LDFLAGS}" -o /out/pgway-cp ./cmd/pgway-cp \
	&& go build -trimpath -ldflags="${LDFLAGS}" -o /out/pgway-dp ./cmd/pgway-dp \
	&& go build -trimpath -ldflags="${LDFLAGS}" -o /out/pgctl ./cmd/pgctl

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/pgway /usr/local/bin/pgway
COPY --from=builder /out/pgway-cp /usr/local/bin/pgway-cp
COPY --from=builder /out/pgway-dp /usr/local/bin/pgway-dp
COPY --from=builder /out/pgctl /usr/local/bin/pgctl

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/pgway"]
