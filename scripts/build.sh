#!/bin/sh
set -ex

#go build ./...
go install ./cmd/gdoc-export
go test .
