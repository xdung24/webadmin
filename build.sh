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

# Generate Go documentation (Swagger)
echo "Generating Go documentation..."
swag init --generalInfo main.go --output docs

# Generate a secure JWT secret at build time
echo "Generating JWT secret..."
JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || python3 -c "import secrets; print(secrets.token_urlsafe(32))" 2>/dev/null || node -e "console.log(require('crypto').randomBytes(32).toString('base64url'))" 2>/dev/null)

if [ -z "$JWT_SECRET" ]; then
  echo "Warning: Could not generate JWT secret. Using fallback method..."
  JWT_SECRET=$(date +%s | sha256sum | base64 | head -c 32)
fi

echo "JWT secret generated successfully"

# Build the Go app with the generated JWT secret
echo "Building Go app..."
CGO_ENABLED=1 go build -ldflags "-w -s -X main.jwtSecretString=$JWT_SECRET" -v -o "$BINARY_NAME"

echo "Build completed successfully!"
cp webadmin.bat dist/