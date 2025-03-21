FROM golang:1.24-alpine AS build

WORKDIR /build

RUN apk update && apk add --no-cache musl-dev gcc build-base

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

ARG VERSION
RUN go build -ldflags "-X github.com/robherley/gw-bot/internal/meta.Version=${VERSION}"

FROM alpine

COPY --from=build /build/gw-bot /usr/bin/gw-bot
RUN apk add --no-cache tzdata
ENV TZ=Etc/UTC

ENTRYPOINT [ "/usr/bin/gw-bot" ]
