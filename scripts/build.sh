#!/bin/bash

# Default binary name
binary_name="omnihance-a3-agent"

# Check if a binary name is provided as an argument
if [ $# -ge 1 ]; then
    binary_name=$1
    # Strip quotes from the binary name
    binary_name=$(echo "$binary_name" | tr -d '"')
fi

# Set version (default to dev if not provided)
version="dev"
if [ $# -ge 2 ]; then
    version=$2
    # Strip quotes from the version
    version=$(echo "$version" | tr -d '"')
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

# Set the Go environment variables for building for Windows
export GOARCH=amd64
export GOOS=windows

# Build for Windows
echo "Building $binary_name for Windows (version: $version)..."
go build -ldflags="-w -s -X main.version=$version" -o "bin/${binary_name}/${binary_name}.exe" "cmd/${binary_name}/main.go"
if [ $? -ne 0 ]; then
    echo "Go build for Windows failed!"
    exit 1
fi

# Reset Go environment variables to their defaults
export GOARCH=
export GOOS=

# Build for Linux
echo "Building $binary_name for Linux (version: $version)..."
go build -ldflags="-w -s -X main.version=$version" -o "bin/${binary_name}/${binary_name}" "cmd/${binary_name}/main.go"
if [ $? -ne 0 ]; then
    echo "Go build for Linux failed!"
    exit 1
fi

echo "$binary_name build complete!"
