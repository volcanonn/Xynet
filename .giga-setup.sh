#!/bin/bash
# Install Golang
apt-get update && apt-get install -y golang-go gcc g++ make
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# The AI will now inherit this exact PATH!
export PATH="$PATH:$(go env GOPATH)/bin"

# If you need vue-tsc, you can just add:
npm install -g vue-tsc
