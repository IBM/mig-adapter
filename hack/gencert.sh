
# Gen CA key
openssl genrsa -out ca.key 2048

# Gen CA Cert
openssl req -new -x509 -days 365 -key ca.key \
  -subj "/C=AU/CN=simple-kubernetes-webhook"\
  -out ca.crt

# Gen webhook server key and csr
openssl req -newkey rsa:2048 -nodes -keyout server.key \
  -subj "/C=AU/CN=simple-kubernetes-webhook" \
  -out server.csr

# Gen / Sign webhook server cert for service (default/migadapter) in cluster
openssl x509 -req \
  -extfile <(printf "subjectAltName=DNS:migadapter.default.svc") \
  -days 365 \
  -in server.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt

# Gen / Sign webhook server cert for dev (host ip 192.168.2.14) outside cluster
openssl x509 -req \
  -extfile <(printf "subjectAltName=IP:192.168.2.14") \
  -days 365 \
  -in server.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out tls.crt

# Server.key is tls.key
cp server.key tls.key

# Gen secret for webhook
kubectl create secret tls simple-kubernetes-webhook-tls \
  --cert=server.crt \
  --key=server.key \
  --dry-run=client -o yaml \
  > webhook.tls.secret.yaml

# Get ca bundle
cat ca.crt | base64 

# Manual steps: fill ca bundle into MutatingWebhookConfiguration
