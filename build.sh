#!/bin/bash
set -e

echo "Generating Prisma client..."
go run github.com/steebchen/prisma-client-go generate

echo "Building application..."
go build -tags netgo -ldflags '-s -w' -o app