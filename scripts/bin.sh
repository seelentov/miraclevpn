#!/bin/bash

go build -o bin/api cmd/main/main.go

./scripts/bin_services.sh