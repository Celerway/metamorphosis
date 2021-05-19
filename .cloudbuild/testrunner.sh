#!/bin/sh
set -x
set -e
wget https://github.com/vectorizedio/redpanda/releases/download/v21.5.2/rpk-linux-amd64.zip
unzip rpk-linux-amd64.zip
mv rpk /usr/local/bin
rpk topic list
go install github.com/pingcap/failpoint/failpoint-ctl@latest
/go/bin/failpoint-ctl enable
go test ./...
/go/bin/failpoint-ctl disable
