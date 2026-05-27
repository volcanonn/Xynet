#!/bin/bash
# Install Golang
apt-get update && apt-get install -y golang-go gcc g++ make
# Install Wails
go install github.com/wailsapp/wails/v2/cmd/wails@latest
export PATH="$PATH:$(go env GOPATH)/bin"
