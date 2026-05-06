# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands
- `go build ./...` — Build all packages
- `go test ./...` — Run all tests
- `go test -v -run TestName ./...` — Run a specific test
- `go vet ./...` — Run vet checks
- `gofmt -w .` — Format all files
