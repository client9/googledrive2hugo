#!/bin/sh
set -ex
./scripts/godownloader-goreleaser.sh

# gometalinter
# https://github.com/alecthomas/gometalinter#binary-releases
curl -L https://git.io/vp6lP | sh
