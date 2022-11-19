FROM golang:1.19.3-alpine3.16 AS builder
WORKDIR /app
# We download these to ensure that when building with CGO, stdlib links are in the right place for alpine
RUN apk update
RUN apk upgrade
RUN apk add --update go gcc g++
COPY . .
RUN go mod download
# TODO: Try CGO-less sqlite connector
RUN GOOS=linux go build -v -o gunslinger

FROM alpine:3.16 as tailscale
WORKDIR /app
COPY . ./
ENV TSFILE=tailscale_1.22.2_amd64.tgz
RUN wget https://pkgs.tailscale.com/stable/${TSFILE} && tar xzf ${TSFILE} --strip-components=1
COPY . ./

FROM alpine:3.16
RUN apk update && apk add ca-certificates iptables ip6tables && rm -rf /var/cache/apk/*

# Copy binary to production image
COPY --from=builder /app/gunslinger /app/gunslinger
COPY --from=builder /app/start.sh /app/start.sh
COPY --from=tailscale /app/tailscaled /app/tailscaled
COPY --from=tailscale /app/tailscale /app/tailscale
RUN mkdir -p /var/run/tailscale /var/cache/tailscale /var/lib/tailscale

ENV PORT=8080
EXPOSE 8080

# Run on container startup.
CMD ["/app/start.sh"]
