FROM golang:1.18

RUN mkdir -p /opt/vacuum

WORKDIR /opt/vacuum

COPY . ./

RUN go mod download && go mod verify
RUN go build -v -o /vacuum vacuum.go

FROM debian:bullseye-slim
WORKDIR /work
COPY --from=0 /vacuum /

ENTRYPOINT ["/vacuum"]
