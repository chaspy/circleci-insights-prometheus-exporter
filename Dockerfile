FROM golang:1.16.5 as builder

WORKDIR /go/src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64
RUN go build \
    -o /go/bin/circleci-insights-prometheus-exporter \
    -ldflags '-s -w'

FROM alpine:3.14.2 as runner

COPY --from=builder /go/bin/circleci-insights-prometheus-exporter /app/circleci-insights-prometheus-exporter

RUN adduser -D -S -H exporter

USER exporter

ENTRYPOINT ["/app/circleci-insights-prometheus-exporter"]
