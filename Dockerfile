FROM golang:1.24.1-bookworm AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y libvorbis-dev libogg-dev

COPY . .
RUN go mod download
RUN GOOS=linux go build -ldflags "-s -w" -v -o gunslinger

# Based on Debian but includes a minimal headless chrome
FROM chromedp/headless-shell:latest
RUN apt-get update && apt-get install -y ca-certificates iptables procps sqlite3 libvorbis-dev libogg-dev && rm -rf /var/lib/apt/lists/*

# Copy binary to production image
COPY --from=builder /app/gunslinger /app/gunslinger

ENV PORT=8080
EXPOSE 8080

# Run on container startup.
ENTRYPOINT ["/app/gunslinger"]