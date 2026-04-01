#!/bin/bash
set -euo pipefail

# Setup Let's Encrypt for tewodros-terminal
# Run on the server as root: sudo bash scripts/setup-letsencrypt.sh

DOMAIN="tewodros.me"
CERT_DIR="/opt/tewodros-terminal/certs"
SERVICE="tewodros-terminal"

echo "==> Installing certbot..."
apt-get update -qq
apt-get install -y -qq certbot

echo "==> Stopping $SERVICE to free port 443..."
systemctl stop "$SERVICE"

echo "==> Obtaining certificate for $DOMAIN..."
certbot certonly --standalone \
  --non-interactive \
  --agree-tos \
  --preferred-challenges tls-alpn-01 \
  --email assefa@tewodros.me \
  -d "$DOMAIN"

echo "==> Setting up certificate directory..."
mkdir -p "$CERT_DIR"

# Symlink so the service always reads the latest cert
ln -sf "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" "$CERT_DIR/cert.pem"
ln -sf "/etc/letsencrypt/live/$DOMAIN/privkey.pem" "$CERT_DIR/key.pem"

# Let the deploy user read the certs
chmod 755 /etc/letsencrypt/live
chmod 755 /etc/letsencrypt/archive
chmod 644 "/etc/letsencrypt/archive/$DOMAIN/privkey"*.pem

echo "==> Starting $SERVICE..."
systemctl start "$SERVICE"

echo "==> Setting up auto-renewal hook..."
cat > /etc/letsencrypt/renewal-hooks/deploy/restart-tewodros.sh << 'EOF'
#!/bin/bash
chmod 644 /etc/letsencrypt/archive/tewodros.me/privkey*.pem
systemctl restart tewodros-terminal
EOF
chmod +x /etc/letsencrypt/renewal-hooks/deploy/restart-tewodros.sh

echo "==> Testing renewal..."
certbot renew --dry-run

echo "==> Done! Certificate installed at:"
echo "    Cert: $CERT_DIR/cert.pem"
echo "    Key:  $CERT_DIR/key.pem"
