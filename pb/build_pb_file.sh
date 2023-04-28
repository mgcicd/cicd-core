#!/bin/bash
#传入一个路由 比如 sh build_pb_file.sh v1/monitor/role/role.proto
serviceProtoPath=$1

#--proto_path 从哪里读取.proto文件
#--go_out 生成GO 代码 到哪个路径下
#plugins=grpc: 这个是 生成client和server插件
protoc --proto_path=. --go_out=plugins=grpc:. ${serviceProtoPath}