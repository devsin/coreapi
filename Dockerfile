# Builder
FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/coreapi ./cmd/coreapi

# Runtime
FROM alpine:3.23.2
RUN apk add --no-cache ca-certificates
WORKDIR /srv

COPY --from=builder /out/coreapi /usr/local/bin/coreapi
COPY db/geocity/GeoLite2-City.mmdb /srv/db/geocity/GeoLite2-City.mmdb

ENV PORT=8080
ENV GEOIP_DB_PATH=/srv/db/geocity/GeoLite2-City.mmdb
EXPOSE 8080

CMD ["/usr/local/bin/coreapi"]
