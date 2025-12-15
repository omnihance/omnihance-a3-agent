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

# Set the Go environment variables for building for Windows
export GOARCH=amd64
export GOOS=windows

# Build for Windows
echo "Building $binary_name for Windows (version: $version)..."
go build -ldflags="-w -s -X main.version=$version" -o "bin/${binary_name}/${binary_name}.exe" "cmd/${binary_name}/main.go"

# Reset Go environment variables to their defaults
export GOARCH=
export GOOS=

# Build for Linux
echo "Building $binary_name for Linux (version: $version)..."
go build -ldflags="-w -s -X main.version=$version" -o "bin/${binary_name}/${binary_name}" "cmd/${binary_name}/main.go"

echo "$binary_name build complete!"
