#!/bin/bash

set -e

GROUPS_VERSION="master"

curl -o proto/groups.proto "https://raw.githubusercontent.com/SPooNqir/groups/$GROUPS_VERSION/proto/groups.proto"

GEN_PATH="."
GO_LIB_PATH=$(go env GOPATH)/src
GOPATH=$(go env GOPATH)

mkdir -p proto/google/api 
curl -o proto/google/api/annotations.proto "https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto"
curl -o proto/google/api/http.proto "https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto"

mkdir -p proto/google/protobuf
curl -o proto/google/protobuf/descriptor.proto "https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/descriptor.proto"
curl -o proto/google/protobuf/empty.proto "https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/empty.proto"
curl -o proto/google/protobuf/struct.proto "https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/struct.proto"
curl -o proto/google/protobuf/timestamp.proto "https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/timestamp.proto"
curl -o proto/google/protobuf/any.proto "https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/any.proto"

echo "gen from proto"

protoc \
        -I proto \
        -I $GO_LIB_PATH/include \
        --go_out=$GEN_PATH      --go_opt=paths=source_relative \
        --go-grpc_out=$GEN_PATH --go-grpc_opt=paths=source_relative \
        --grpc-gateway_out=logtostderr=true,paths=source_relative:$GEN_PATH \
        --openapiv2_out=logtostderr=true:. \
        proto/users.proto

protoc-go-inject-tag -input=$GEN_PATH/users.pb.go

echo "get mod"
go mod tidy
go get -u ./...
