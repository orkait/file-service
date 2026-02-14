#!/bin/bash
# Build script for file-service

set -e

echo "Building file-service..."

# Clean previous builds
rm -f file-service file-service.exe

# Build for current platform
go build -o file-service .

echo "Build complete: file-service"
