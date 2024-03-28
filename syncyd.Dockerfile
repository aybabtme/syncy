ARG BASE_BUILDER_IMAGE=golang:1.22.1
ARG BASE_SERVER_IMAGE=debian:stable-slim

FROM ${BASE_BUILDER_IMAGE} AS builder

WORKDIR /usr/src/app
COPY . .
RUN go mod download && go mod verify
RUN go build -v -o /usr/local/bin/syncyd ./cmd/syncyd

FROM ${BASE_SERVER_IMAGE} AS server
COPY --from=builder /usr/local/bin/syncyd /usr/local/bin/syncyd
ENTRYPOINT ["/usr/local/bin/syncyd"]