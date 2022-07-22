#!/bin/bash +x

if [ -n "$1" ]; then
    VERSION=$1
else
    echo 'Version is not set'
    exit 1
fi

echo "Upgrading libraries to version: v$VERSION"

IFS=. VERSION_PARTS=($VERSION) IFS=' '
MAJOR=${VERSION_PARTS[0]}

echo "Major libraries version: v$MAJOR"

go get github.com/specgen-io/specgen/v$MAJOR@v$VERSION

go build