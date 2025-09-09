#!/usr/bin/env bash

echo "Installing BuildRequires dependencies";echo
grep ^BuildRequires dvol.spec |awk -F\: '{print "sudo dnf install -y"$2}'|sed -e 's/,/ /g' | sh
echo;echo;echo "Done. Now installing the Go binaries"
#/opt/bin/install_golang.sh $(grep ^go src/go.mod | awk '{print $2}') amd64


export VER=`cat go.version`
export ARCH=${1:-"amd64"}

echo "Fetching archive..."
sudo wget -q https://go.dev/dl/go${VER}.linux-${ARCH}.tar.gz -O /tmp/go.tar.gz -O /opt/go.tar.gz

echo "Unarchiving..."
cd /opt ; sudo rm -rf go;sudo tar zxf go.tar.gz; sudo rm -f go.tar.gz

echo "Completed."

