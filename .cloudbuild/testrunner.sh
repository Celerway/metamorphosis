#!/bin/sh

# This runs the tests inside of Google Cloud Build.
set -x
set -e
apt-get update
apt-get install -y unzip
wget -q https://github.com/vectorizedio/redpanda/releases/download/v21.5.2/rpk-linux-amd64.zip
unzip rpk-linux-amd64.zip
mv rpk /usr/local/bin
rpk topic --brokers 127.0.0.1:9092 list
go install github.com/pingcap/failpoint/failpoint-ctl@latest
/go/bin/failpoint-ctl enable
go test ./...
/go/bin/failpoint-ctl disable
