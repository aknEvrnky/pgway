# syntax=docker/dockerfile:1

FROM golang:1.27-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN mkdir -p /out \
	&& go build -trimpath -ldflags="-s -w" -o /out/pgway ./cmd/pgway \
	&& go build -trimpath -ldflags="-s -w" -o /out/pgway-cp ./cmd/pgway-cp \
	&& go build -trimpath -ldflags="-s -w" -o /out/pgway-dp ./cmd/pgway-dp \
	&& go build -trimpath -ldflags="-s -w" -o /out/pgctl ./cmd/pgctl

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/pgway /usr/local/bin/pgway
COPY --from=builder /out/pgway-cp /usr/local/bin/pgway-cp
COPY --from=builder /out/pgway-dp /usr/local/bin/pgway-dp
COPY --from=builder /out/pgctl /usr/local/bin/pgctl

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/pgway"]
