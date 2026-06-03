#!/bin/bash

# Default binary name
binary_name="omnihance-a3-agent"

# Check if a binary name is provided as an argument
if [ $# -eq 1 ]; then
    binary_name=$1
fi

# Build UI first
echo "Building UI..."
cd "cmd/${binary_name}/${binary_name}-ui" || exit 1
pnpm run build
if [ $? -ne 0 ]; then
    echo "UI build failed!"
    exit 1
fi
cd ../../..

# Run the Go application
go run "cmd/${binary_name}/main.go"
