FROM golang:1.20.3-alpine3.16 AS builder

WORKDIR /app

COPY . .
RUN go mod download
RUN GOOS=linux CGO_ENABLED=0 go build -v -o gunslinger

FROM alpine:3.17
RUN apk update && apk add ca-certificates iptables ip6tables && rm -rf /var/cache/apk/*

# Copy binary to production image
COPY --from=builder /app/gunslinger /app/gunslinger

ENV PORT=8080
EXPOSE 8080

# Run on container startup.
CMD ["/app/gunslinger"]
