#!/usr/bin/env bash

set -e

function main() {
  if ! type >/dev/null 2>&1 nex; then
    echo "# 'nex' command not found"
    echo "# run below command"
    echo "go get -u github.com/blynn/nex"
    return 1
  fi
  if ! type >/dev/null 2>&1 goyacc; then
    echo "# 'goyacc' command not found"
    echo "# run below command"
    echo "go get -u golang.org/x/tools/cmd/goyacc"
    return 1
  fi
  echo '[nex]'
  nex lexer.nex
  echo '[goyacc]'
  goyacc -o paser.y.go -v parser.y.output parser.y
  go build -o goenkins-format
}
main "$@"
