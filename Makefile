CONFIG_PATH=${HOME}/.dislog/

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

#----------------------------------------
# generate certificate

.PHONY: gencert-init
gencert-init:
	cfssl gencert \
		-initca certs/ca-csr.json | cfssljson -bare ca

.PHONY: gencert-server
gencert-server:
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certs/ca-config.json \
		-profile=server \
		certs/server-csr.json | cfssljson -bare server
	
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: gencert-client
gencert-client:
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certs/ca-config.json \
		-profile=client \
		certs/client-csr.json | cfssljson -bare client
	
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: gencert-multi-client
gencert-multi-client:
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certs/ca-config.json \
		-profile=client \
		-cn="root" \
		certs/client-csr.json | cfssljson -bare root-client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=certs/ca-config.json \
		-profile=client \
		-cn="nobody" \
		certs/client-csr.json | cfssljson -bare nobody-client
	
	mv *.pem *.csr ${CONFIG_PATH}

# generate certificate
.PHONY: gencert-all
gencert-all:
	cfssl print-defaults csr > certs/ca-csr.json
	cfssl print-defaults config > certs/ca-config.json

	make gencert-init
	make gencert-server
	make gencert-client

#----------------------------------------

# START: compile
.PHONY: compile
compile:
	protoc api/v1/*.proto \
		--go_out=. \
		--go-grpc_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		--proto_path=.
# END: compile

test:
	go test -race ./...
