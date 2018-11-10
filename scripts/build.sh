#!/bin/sh
set -ex
export GO111MODULE=off
go get ./...
go test .
go install ./cmd/gdoc-export
