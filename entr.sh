#!/bin/bash
echo "m2.go" | entr -r bash -c 'while true ; do clear ; set -x ; go get -x -u ./... ; go mod tidy ; go mod vendor ; set +x ; (go run -x . --help && go run -x . run --help && MENV=m2.conf go run -x . run) || for i in {10..1}; do s="${i}..."; d=$(awk "BEGIN{print 0.95/${#s}}"); for ((j=0;j<${#s};j++)); do printf "%s" "${s:j:1}"; sleep "$d"; done; done; done'
