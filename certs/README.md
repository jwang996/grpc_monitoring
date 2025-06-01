# Generate mTLS

## 1) Create a certs/ folder if it doesn't exist (you can skip this since I've prepared the materials)
`mkdir -p certs`

`cd certs`

## 2) Generate a CA private key and self-signed cert (10 years)
`openssl genrsa -out ca.key.pem 4096
openssl req -x509 -new -nodes \
-key ca.key.pem \
-sha256 \
-days 3650 \
-out ca.crt.pem \
-subj "/C=US/ST=Test/L=Test/O=MyOrg/OU=TestCA/CN=Test Root CA"`

## 3) Generate server key + CSR
`openssl genrsa -out server.key.pem 4096`

`openssl req -new \
-key server.key.pem \
-out server.csr.pem \
-subj "/C=US/ST=Test/L=Test/O=MyOrg/OU=Server/CN=localhost"`

## 4) Create server_ext.cnf for SAN=localhost

```
cat > server_ext.cnf << 'EOF'
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
EOF
```

## 5) Sign server CSR with CA
`openssl x509 -req \
-in server.csr.pem \
-CA ca.crt.pem \
-CAkey ca.key.pem \
-CAcreateserial \
-out server.crt.pem \
-days 365 \
-sha256 \
-extfile server_ext.cnf`

## 6) Generate client key + CSR

`openssl genrsa -out client.key.pem 4096`

`openssl req -new \
-key client.key.pem \
-out client.csr.pem \
-subj "/C=US/ST=Test/L=Test/O=MyOrg/OU=Client/CN=Test Client"`

## 7) Create client_ext.cnf for clientAuth
```
cat > client_ext.cnf << 'EOF'
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = clientAuth
EOF
```

## 8) Sign client CSR with CA
`openssl x509 -req \
-in client.csr.pem \
-CA ca.crt.pem \
-CAkey ca.key.pem \
-CAcreateserial \
-out client.crt.pem \
-days 365 \
-sha256 \
-extfile client_ext.cnf`

## (Optional cleanup of CSRs & .srl)
`rm *.csr.pem ca.srl
`
## 9) Secure your keys (optional)
`chmod 600 *.key.pem
`
## Show resulting files:
`ls -1`

* ca.crt.pem
* ca.key.pem
* client.crt.pem
* client_ext.cnf
* client.key.pem
* server.crt.pem
* server_ext.cnf
* server.key.pem