FROM golang:alpine as yd-server
ENV GO111MODULE=on
WORKDIR /server
COPY go.mod go.sum /server/
RUN go mod download

ADD . .
RUN go build -o bin/yd-server cmd/main.go

FROM alpine:latest

WORKDIR /
RUN apk add --no-cache tzdata
COPY --from=yd-server /server/bin .
COPY --from=yd-server /server/database/migrations ./database/migrations

EXPOSE 8080
ENTRYPOINT ["./yd-server"]
