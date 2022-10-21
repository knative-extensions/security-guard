# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.18 AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /guard-rproxy cmd/guard-rproxy/*.go


##
## Deploy
##
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /guard-rproxy /guard-rproxy

EXPOSE 22000

USER nonroot:nonroot

ENTRYPOINT ["/guard-rproxy"]
