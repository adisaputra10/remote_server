#!/bin/bash
# Generate self-signed certificates for sh.adisaputra.online

set -e

DOMAIN="sh.adisaputra.online"
CERT_DIR="certs"
CERT_FILE="$CERT_DIR/server.crt"
KEY_FILE="$CERT_DIR/server.key"
CSR_FILE="$CERT_DIR/server.csr"

echo "========================================"
echo "Self-Signed Certificate Generator"
echo "========================================"
echo "Domain: $DOMAIN"
echo "Certificate Directory: $CERT_DIR"
echo "========================================"

# Create certificates directory
mkdir -p "$CERT_DIR"
cd "$CERT_DIR"

echo "Generating private key..."
openssl genrsa -out server.key 2048
chmod 600 server.key

echo "Creating certificate signing request..."
cat > server.conf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C=ID
ST=Jakarta
L=Jakarta
O=Remote Tunnel
OU=IT Department
CN=$DOMAIN

[v3_req]
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = *.$DOMAIN
DNS.3 = localhost
IP.1 = 127.0.0.1
EOF

echo "Generating certificate signing request..."
openssl req -new -key server.key -out server.csr -config server.conf

echo "Generating self-signed certificate (valid for 365 days)..."
openssl x509 -req -in server.csr -signkey server.key -out server.crt -days 365 -extensions v3_req -extfile server.conf

# Set proper permissions
chmod 644 server.crt
chmod 600 server.key

echo
echo "✅ Self-signed certificate generated successfully!"
echo
echo "Files created:"
echo "- Private Key: $(pwd)/server.key"
echo "- Certificate: $(pwd)/server.crt"
echo "- CSR: $(pwd)/server.csr"
echo "- Config: $(pwd)/server.conf"
echo
echo "Certificate Details:"
openssl x509 -in server.crt -text -noout | grep -A 1 "Subject:"
openssl x509 -in server.crt -text -noout | grep -A 3 "Subject Alternative Name"
openssl x509 -in server.crt -text -noout | grep -A 2 "Validity"

echo
echo "To use these certificates:"
echo "1. Update .env.production:"
echo "   CERT_FILE=$PWD/server.crt"
echo "   KEY_FILE=$PWD/server.key"
echo
echo "2. Start relay server:"
echo "   ./start-relay.sh"
echo
echo "3. Test certificate:"
echo "   openssl s_client -connect $DOMAIN:8443 -servername $DOMAIN"
echo
echo "⚠️  Note: Clients will need to accept self-signed certificate"
echo "   or add -k flag to curl commands"

cd ..
