#!/bin/bash

go test ./... -coverprofile coverage.out
go tool cover -html=coverage.out -o coverage.html
result=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo "Total coverage: $result"