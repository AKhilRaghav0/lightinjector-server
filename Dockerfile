FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY proxy/ .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -o proxy .

FROM alpine:3.19
RUN apk add --no-cache openssh shadow bash && \
    adduser -D -s /bin/bash tunnel && \
    ssh-keygen -A && \
    mkdir -p /run/sshd

RUN printf 'Port 2222\nPermitRootLogin no\nPasswordAuthentication yes\nPubkeyAuthentication yes\nAllowTcpForwarding yes\nGatewayPorts no\nX11Forwarding no\nPrintMotd no\nClientAliveInterval 30\nClientAliveCountMax 3\nMaxSessions 100\n' > /etc/ssh/sshd_config

COPY --from=builder /build/proxy /usr/local/bin/proxy
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 8080
CMD ["/entrypoint.sh"]
