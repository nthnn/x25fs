#!/usr/bin/env bash
set -euo pipefail

VERSION="1.0.0"
PACKAGE_NAME="x25fs"
BIN_NAME="x25fs"
DIST_DIR="dist"
PKG_DIR="${DIST_DIR}/${PACKAGE_NAME}-${VERSION}-pkg"
INSTALL_PATH="/usr/bin"

mkdir -p "${DIST_DIR}"

echo "Building ${PACKAGE_NAME}..."
go build -ldflags "-s -w" -o "${DIST_DIR}/${BIN_NAME}"

echo "Setting up package directory..."
rm -rf "${PKG_DIR}"
mkdir -p "${PKG_DIR}${INSTALL_PATH}"
mkdir -p "${PKG_DIR}/DEBIAN"

cp "${DIST_DIR}/${BIN_NAME}" "${PKG_DIR}${INSTALL_PATH}/"
cat > "${PKG_DIR}/DEBIAN/control" <<EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: $(dpkg --print-architecture)
Maintainer: Nathanne Isip (nathanneisip@gmail.com)
Description: x25fs is a FUSE (Filesystem in Userspace)
  implementation that provides an encrypted and secure
  file storage solution.
EOF

chmod 0755 "${PKG_DIR}/DEBIAN"
chmod 0755 "${PKG_DIR}${INSTALL_PATH}/${BIN_NAME}"

echo "Building Debian package..."
dpkg-deb --build "${PKG_DIR}" "${DIST_DIR}/${PACKAGE_NAME}-${VERSION}.deb"

rm -rf "${PKG_DIR}"
echo "Created ${DIST_DIR}/${PACKAGE_NAME}-${VERSION}.deb"
