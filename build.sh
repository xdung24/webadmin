#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Build the Angular app using Yarn
echo "Building Angular app..."
yarn build

# Determine the output binary name based on the OS
if [[ "$(uname -s)" == "MINGW"* || "$(uname -s)" == "CYGWIN"* || "$(uname -s)" == "MSYS"* ]]; then
  BINARY_NAME="dist/webadmin.exe"
else
  BINARY_NAME="dist/webadmin"
fi

# Build the Go app
echo "Building Go app..."
go build -o "$BINARY_NAME"

echo "Build completed successfully!"