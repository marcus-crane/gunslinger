FROM golang:1.17 AS builder
WORKDIR /go/src/app
COPY . .
RUN go mod download
RUN GOOS=linux go build -v -o app

FROM alpine:latest
COPY --from=builder /go/src/app/app /goapp/app
WORKDIR /goapp
RUN apk --no-cache add ca-certificates
EXPOSE 8080

CMD ["/goapp/app"]
