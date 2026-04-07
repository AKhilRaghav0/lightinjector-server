FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY proxy/ .
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o proxy .

FROM alpine:3.19
RUN apk add --no-cache openssh bash && \
    adduser -D -s /bin/bash tunnel && \
    ssh-keygen -A

COPY --from=builder /build/proxy /usr/local/bin/proxy
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN cat > /etc/ssh/sshd_config << 'EOF'
Port 2222
PermitRootLogin no
PasswordAuthentication yes
PubkeyAuthentication yes
AllowTcpForwarding yes
GatewayPorts no
X11Forwarding no
PrintMotd no
ClientAliveInterval 30
ClientAliveCountMax 3
MaxSessions 100
EOF

EXPOSE 8080
CMD ["/entrypoint.sh"]
