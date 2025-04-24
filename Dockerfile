FROM golang:1.23.1 AS build

RUN apt-get update && apt-get install -y \
    libvips-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .
RUN ls -lah /app
RUN go build -o chrononewsapi cmd/app/main.go
RUN ls -lah /app

FROM alpine:3.20.2

RUN apk add --no-cache tzdata \
    vips \
    glib \
    gobject-introspection \
    libc6-compat

RUN apk add --no-cache tzdata

ENV TZ=UTC

RUN apk add --no-cache libc6-compat

WORKDIR /app

COPY --from=build /app/chrononewsapi /app/chrononewsapi

CMD ["/app/chrononewsapi"]
