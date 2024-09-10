FROM golang:1.23

RUN mkdir -p /opt/vacuum

WORKDIR /opt/vacuum

COPY . ./

RUN go mod download && go mod verify
RUN go build -ldflags="-w -s" -v -o /vacuum vacuum.go

FROM debian:bookworm-slim

WORKDIR /work

COPY --from=0 /vacuum /usr/local/bin/vacuum

COPY docker-entrypoint.sh /

ENTRYPOINT ["/docker-entrypoint.sh"]
