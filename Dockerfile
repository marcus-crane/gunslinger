FROM golang:1.21.3-alpine3.18 AS builder

WORKDIR /app

COPY . .
RUN go mod download
RUN GOOS=linux CGO_ENABLED=0 go build -v -o gunslinger

FROM alpine:3.18
RUN apk update && apk add ca-certificates iptables ip6tables sqlite && rm -rf /var/cache/apk/*

# Copy binary to production image
COPY --from=builder /app/gunslinger /app/gunslinger

ENV PORT=8080
EXPOSE 8080

# Run on container startup.
CMD ["/app/gunslinger"]
