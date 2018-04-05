#!/bin/sh
set -ex

#go build ./...
go test .
go install ./cmd/gdoc-export
