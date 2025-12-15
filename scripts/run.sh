#!/bin/bash

# Default binary name
binary_name="omnihance-a3-agent"

# Check if a binary name is provided as an argument
if [ $# -eq 1 ]; then
    binary_name=$1
fi

go run "cmd/${binary_name}/main.go"
