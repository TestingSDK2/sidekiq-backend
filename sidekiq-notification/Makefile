.PHONY: proto

# Default version is 1
VERSION ?= 1

proto:
	protoc protobuf/v$(VERSION)/*.proto --go_out=../sidekiq-proto --go-grpc_out=../sidekiq-proto

### TO RUN make proto VERSION=2