#!/bin/sh
# Installs toxiproxy in the container
set -e
ARCH=$(uname -m)
TOXIPROXY_VERSION=v2.4.0

case $ARCH in
  x86_64)
    ARCH=amd64
    ;;
  aarch64)
    ARCH=arm64
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac
echo "Installing toxiproxy for $ARCH"
URL="https://github.com/Shopify/toxiproxy/releases/download/$TOXIPROXY_VERSION/toxiproxy-server-linux-$ARCH"
echo "Downloading $URL"
wget -O /bin/toxiproxy-server $URL
chmod +x /bin/toxiproxy-server