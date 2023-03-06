CONFIG_PATH=${HOME}/.dislog/

.PHONY: init
init:
	mkdir -p ${CONFIG_PATH}

#----------------------------------------
# generate certificate

.PHONY: gencert-init
gencert-init:
	cfssl gencert \
		-initca test/ca-csr.json | cfssljson -bare ca

.PHONY: gencert-server
gencert-server:
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=server \
		test/server-csr.json | cfssljson -bare server
	
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: gencert-client
gencert-client:
	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		test/client-csr.json | cfssljson -bare client
	
	mv *.pem *.csr ${CONFIG_PATH}

.PHONY: gencert-multi-client
gencert-multi-client:
cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="root" \
		test/client-csr.json | cfssljson -bare root-client

	cfssl gencert \
		-ca=ca.pem \
		-ca-key=ca-key.pem \
		-config=test/ca-config.json \
		-profile=client \
		-cn="nobody" \
		test/client-csr.json | cfssljson -bare nobody-client
	
	mv *.pem *.csr ${CONFIG_PATH}

# generate certificate
.PHONY: gencert-all
gencert-all:
	make gencert-init
	make gencert-server
	make gencert-client

#----------------------------------------

# START: compile
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
