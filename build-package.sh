#!/bin/bash

set -e

PACKAGE_DIR="package"
TARGET_DIR="dependencies-src"
REPO_URL="https://github.com/open-edge-platform/orch-cli.git"
REPO_NAME="orch-cli"
CLONE_DIR="source-$REPO_NAME"

mkdir -p "$TARGET_DIR"

make build

sudo rm -rf "$PACKAGE_DIR"
mkdir "$PACKAGE_DIR"
mkdir "$PACKAGE_DIR/$CLONE_DIR"
mkdir "$PACKAGE_DIR/$TARGET_DIR"

cp build/_output/orch-cli "$PACKAGE_DIR/"
cp go.mod go.sum "$PACKAGE_DIR/"

echo "Cloning your repository from $REPO_URL..."
git clone "$REPO_URL" "$PACKAGE_DIR/$CLONE_DIR"

echo "Extracting dependencies from go.mod..."

cd "$PACKAGE_DIR"
while read -r mod ver; do
    # Skip empty lines or lines starting with #
    [[ -z "$mod" || "$mod" =~ ^# ]] && continue
    echo "Package: $mod"
    echo "Version: $ver"
    go mod download "$mod@$ver"
    modpath=$(go list -f '{{.Dir}}' -m "$mod@$ver")
    if [ -d "$modpath" ]; then
        cp -r --no-preserve=mode,ownership "$modpath" "$TARGET_DIR/"
    fi
done < ../sbom.txt

echo "All sources and repo archive are in $PACKAGE_DIR/"

echo "Creating package archive..."
tar -czf "../orch-cli-package.tar.gz" \
    "orch-cli" \
    "$CLONE_DIR" \
    "$(basename "$TARGET_DIR")"

cd ..

echo "Package created at .$PACKAGE_DIR/orch-cli-package.tar.gz"
