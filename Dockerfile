FROM golang:1.16 AS builder
ENV CGO_ENABLED=0
WORKDIR /app
COPY . /app
RUN go build -v -o smart_exporter

FROM alpine:3.13
RUN apk add --no-cache smartmontools
COPY --from=builder /app/smart_exporter /
CMD ["/smart_exporter"]
EXPOSE 9649
