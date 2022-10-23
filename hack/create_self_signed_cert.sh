// Generate private key
OpenSSL genrsa -out ca.key 2048

// Create a self-signed certificate
openssl req -x509   -new -nodes    -days 365   -key ca.key   -out ca.crt   -subj "/CN=guard-service.default"

// Create the tls secret
kubectl create secret tls guard-service-tls --key ca.key --cert ca.crt

