#!/bin/bash
protoc -I. -I/usr/include/googleapis/ --include_imports --include_source_info --go_out=plugins=grpc:. --descriptor_set_out=health.pb  health.proto