FROM golang:1.24-alpine AS build

WORKDIR /build

RUN apk update && apk add --no-cache musl-dev gcc build-base

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY . .

RUN go build

FROM alpine

COPY --from=build /build/gw-bot /usr/bin/gw-bot

ENTRYPOINT [ "/usr/bin/gw-bot" ]
