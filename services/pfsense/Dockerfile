FROM golang:1.23-bookworm AS build

WORKDIR /src
COPY go.mod ./
COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

FROM cgr.dev/chainguard/static:latest

ARG VERSION=0.0.0
ARG BUILD_DATE
ARG COMMIT_SHA

COPY --from=build /out/server /server

EXPOSE 8080

LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_DATE}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"

ENTRYPOINT ["/server"]
