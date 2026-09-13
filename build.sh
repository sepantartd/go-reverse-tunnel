#!/bin/bash

# Create a bin directory for binaries
mkdir -p bin

echo "[*] Building for Linux (Server/Client)..."
GOOS=linux GOARCH=amd64 go build -o bin/server-linux-amd64 cmd/server/main.go
GOOS=linux GOARCH=amd64 go build -o bin/client-linux-amd64 cmd/client/main.go

echo "[*] Building for Windows (Server/Client)..."
GOOS=windows GOARCH=amd64 go build -o bin/server-windows-amd64.exe cmd/server/main.go
GOOS=windows GOARCH=amd64 go build -o bin/client-windows-amd64.exe cmd/client/main.go

echo "[*] Building for Android (ARM64)..."
GOOS=android GOARCH=arm64 go build -o bin/client-android-arm64 cmd/client/main.go

echo "[+] Build completed successfully! Check the 'bin/' directory."
