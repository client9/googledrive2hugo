#!/bin/sh -e
# autorelease based on tag
if test -z "$TRAVIS_TAG"; then
	echo "no tag found, not goreleasing"
	exit 0
fi
echo "found tag ${TRAVIS_TAG}"
curl -sL https://git.io/goreleaser | bash
