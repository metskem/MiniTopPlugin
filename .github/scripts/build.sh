#!/bin/bash

OUTPUT_DIR=$PWD/dist
mkdir -p "${OUTPUT_DIR}"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${OUTPUT_DIR}"/MiniTopPlugin_linux_amd64 -ldflags "-X github.com/rabobank/MiniTopPlugin/version.VERSION=${VERSION} -X github.com/rabobank/MiniTopPlugin/version.COMMIT=${COMMIT}" .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o "${OUTPUT_DIR}"/MiniTopPlugin_darwin_amd64 -ldflags "-X github.com/rabobank/MiniTopPlugin/version.VERSION=${VERSION} -X github.com/rabobank/MiniTopPlugin/version.COMMIT=${COMMIT}" .
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o "${OUTPUT_DIR}"/MiniTopPlugin_darwin_arm64 -ldflags "-X github.com/rabobank/MiniTopPlugin/version.VERSION=${VERSION} -X github.com/rabobank/MiniTopPlugin/version.COMMIT=${COMMIT}" .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "${OUTPUT_DIR}"/MiniTopPlugin_windows_amd64 -ldflags "-X github.com/rabobank/MiniTopPlugin/version.VERSION=${VERSION} -X github.com/rabobank/MiniTopPlugin/version.COMMIT=${COMMIT}" .
