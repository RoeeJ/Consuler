#!/bin/bash
export CONSUL_TOKEN=84c4752d-1a60-6a94-8db0-8b5f549cb333
go test -coverprofile=cover.out -cover ./...
go tool cover -html=cover.out
