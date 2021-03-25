FROM golang AS builder
ENV CGO_ENABLED=0
WORKDIR /proj
ADD . /proj
RUN go build -v -o smart_exporter

FROM alpine
RUN apk add --no-cache ca-certificates smartmontools
COPY --from=builder /proj/smart_exporter /
CMD ["/smart_exporter", "-listen-addr", ":8000"]
EXPOSE 8000
