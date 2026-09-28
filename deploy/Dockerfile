FROM golang:1.26.0-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-s -w" \
    -o /trip-service \
    ./cmd/trip-service

FROM scratch

COPY --from=builder --chown=65532:65532 /trip-service /trip-service

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/trip-service"]
