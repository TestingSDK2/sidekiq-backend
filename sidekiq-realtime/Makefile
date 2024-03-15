install:
	npm install

run:
	node server.js

dev:
	nodemon server.js

generate_grpc:
	protoc --proto_path=. --go_out=../sidekiq-proto --go-grpc_out=../sidekiq-proto ./protobuf/v1/*.proto

.PHONY: install run generate_grpc
