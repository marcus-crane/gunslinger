FROM golang:1.19.4-alpine3.16 AS builder
WORKDIR /app
# We download these to ensure that when building with CGO, stdlib links are in the right place for alpine
RUN apk update
RUN apk upgrade
RUN apk add --update go gcc g++
COPY . .
RUN go mod download
# TODO: Try CGO-less sqlite connector
RUN GOOS=linux go build -v -o gunslinger

FROM alpine:3.17
RUN apk update && apk add ca-certificates iptables ip6tables && rm -rf /var/cache/apk/*

# Copy binary to production image
COPY --from=builder /app/gunslinger /app/gunslinger

ENV PORT=8080
EXPOSE 8080

# Run on container startup.
CMD ["/app/gunslinger"]
