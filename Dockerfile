FROM --platform=$BUILDPLATFORM golang:1.24 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN mkdir -p /opt/vacuum

WORKDIR /opt/vacuum

COPY . ./

RUN go mod download && go mod verify
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s -X 'main.version=$(git describe --tags --abbrev=0)' -X 'main.date=$(date +%Y-%m-%dT%TZ)'" \
    -v -o vacuum vacuum.go

FROM debian:bookworm-slim

WORKDIR /work

COPY --from=builder /opt/vacuum/vacuum /usr/local/bin/vacuum

COPY docker-entrypoint.sh /

ENTRYPOINT ["/docker-entrypoint.sh"]
