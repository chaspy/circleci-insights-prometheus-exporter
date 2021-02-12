FROM golang:1.15.8 as builder

WORKDIR /go/src

COPY go.mod go.sum ./
RUN go mod download

COPY ./main.go  ./

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN go build \
    -o /go/bin/circleci-insight-prometheus-exporter \
    -ldflags '-s -w'

FROM alpine:3.13.1 as runner

COPY --from=builder /go/bin/circleci-insight-prometheus-exporter /app/circleci-insight-prometheus-exporter

RUN adduser -D -S -H exporter

USER exporter

ENTRYPOINT ["/app/circleci-insight-prometheus-exporter"]
