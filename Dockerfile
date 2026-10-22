FROM node:26.12.1-trixie-slim AS ui-builder

WORKDIR /opt/vacuum

COPY . ./

RUN ./scripts/build-ui-assets.sh

FROM --platform=$BUILDPLATFORM golang:1.27.3 AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

RUN mkdir -p /opt/vacuum

WORKDIR /opt/vacuum

COPY --from=ui-builder /opt/vacuum ./

RUN go mod download && go mod verify
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -tags html_report_ui \
    -ldflags="-w -s -X 'main.version=$(git describe --tags --abbrev=0)' -X 'main.date=$(date +%Y-%m-%dT%TZ)'" \
    -v -o vacuum vacuum.go

FROM debian:trixie-slim

WORKDIR /work

COPY --from=builder /opt/vacuum/vacuum /usr/local/bin/vacuum
COPY scripts/docker-entrypoint.sh /docker-entrypoint.sh

ENTRYPOINT ["/docker-entrypoint.sh"]
