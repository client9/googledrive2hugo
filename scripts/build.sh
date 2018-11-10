#!/bin/sh
set -ex
export GO111MODULE=on
go test .
go install ./cmd/gdoc-export
