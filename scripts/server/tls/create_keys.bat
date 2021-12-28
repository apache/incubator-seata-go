openssl genrsa -out ca.key 2048
openssl req -new -key ca.key -out ca.csr
openssl x509 -req -days 3650 -in ca.csr -signkey ca.key -out ca.crt

openssl genrsa -out server.key 2048
openssl req -new -nodes -key server.key -out server.csr -config ./openssl.cnf -extensions v3_req
openssl x509 -req -days 3650 -in server.csr -out server.pem -CA ca.crt -CAkey ca.key -CAcreateserial -extfile ./openssl.cnf -extensions v3_req
