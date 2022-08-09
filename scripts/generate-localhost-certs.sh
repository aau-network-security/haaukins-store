!#/bin/bash

echo "generating root certificate ......\n"
openssl req -x509 -nodes -new -sha256 -days 1024 -newkey rsa:2048 -keyout RootCA.key -out RootCA.pem -subj "/C=US/CN=Haaukins-Root-CA"
openssl x509 -outform pem -in RootCA.pem -out RootCA.crt


echo "creating localhost:50051 certificate ......\n"

openssl req -new -nodes -newkey rsa:2048 -keyout localhost:50051.key -out localhost:50051.csr -subj "/C=DK/ST=Capital/L=Copenhagen/O=Haaukins-Certificates/CN=localhost:50051.local"
openssl x509 -req -sha256 -days 1024 -in localhost:50051.csr -CA RootCA.pem -CAkey RootCA.key -CAcreateserial -extfile domains.ext -out localhost:50051.crt
