#!/bin/sh
PORT=${PORT:-8080}
SSH_PASSWORD=${SSH_PASSWORD:-LightInjector2024!}

# Set password
echo "tunnel:${SSH_PASSWORD}" | chpasswd || echo "WARN: chpasswd failed"

# Optional pubkey auth
if [ -n "$SSH_PUBKEY" ]; then
    mkdir -p /home/tunnel/.ssh
    echo "$SSH_PUBKEY" > /home/tunnel/.ssh/authorized_keys
    chown -R tunnel:tunnel /home/tunnel/.ssh
    chmod 700 /home/tunnel/.ssh
    chmod 600 /home/tunnel/.ssh/authorized_keys
fi

echo "==> Starting sshd on :2222"
/usr/sbin/sshd -f /etc/ssh/sshd_config

echo "==> Starting proxy on :${PORT}"
exec /usr/local/bin/proxy
