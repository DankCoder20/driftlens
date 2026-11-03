FROM golang:1.25-bookworm AS build
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /driftlens ./cmd/server

FROM alpine/helm:3.19.0 AS helm
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates git openssh-client wget && rm -rf /var/lib/apt/lists/* && useradd --uid 10001 --create-home driftlens
COPY --from=helm /usr/bin/helm /usr/local/bin/helm
COPY --from=build /driftlens /usr/local/bin/driftlens
RUN mkdir -p /data /repositories /credentials && chown 10001:10001 /data /repositories /credentials
USER 10001:10001
ENV DRIFTLENS_ADDR=0.0.0.0:8080 DRIFTLENS_DATA_DIR=/data DRIFTLENS_REPOSITORY_ROOT=/repositories DRIFTLENS_CREDENTIAL_ROOT=/credentials
EXPOSE 8080
ENTRYPOINT ["driftlens"]
