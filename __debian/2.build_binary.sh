#!/usr/bin/env bash

PKGDIR="dvol-2.25.00-0_amd64"

mkdir -p ${PKGDIR}/opt/bin ${PKGDIR}/DEBIAN
mkdir -p ${PKGDIR}/opt/bin ${PKGDIR}/DEBIAN
for i in control preinst prerm postinst postrm;do
  mv $i ${PKGDIR}/DEBIAN/
done

echo "Building binary from source"
cd ../src
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o ../__debian/${PKGDIR}/opt/bin/dvol .
sudo chown 0:0 ../__debian/${PKGDIR}/opt/bin/dvol

echo "Binary built. Now packaging..."
cd ../__debian/
dpkg-deb -b ${PKGDIR}
