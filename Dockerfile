FROM golang:1-alpine AS build

WORKDIR /usr/src/app

COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN go mod download

RUN go build -ldflags="-s -w" -o /usr/local/bin/app cmd/awair-offline/main.go

FROM alpine

COPY --from=build /usr/local/bin/app /app

CMD ["/app"]
