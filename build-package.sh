#!/bin/bash

set -e

export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
PACKAGE_DIR="package"
TARGET_DIR="dependencies-src"
RELEASE_VERSION="release-2025.2"
REPO_URL="https://github.com/open-edge-platform/orch-cli/archive/refs/heads/${RELEASE_VERSION}.zip"
REPO_NAME="orch-cli"
CLONE_DIR="source-$REPO_NAME"

sudo rm -rf "$TARGET_DIR"
mkdir -p "$TARGET_DIR"

sudo rm -rf "$PACKAGE_DIR"
mkdir "$PACKAGE_DIR"
mkdir "$PACKAGE_DIR/$CLONE_DIR"
mkdir "$PACKAGE_DIR/$TARGET_DIR"

make build

cp build/_output/orch-cli "$PACKAGE_DIR/"

echo "Downloading your repository from $REPO_URL..."
#wget "$REPO_URL" -O "$PACKAGE_DIR/${RELEASE_VERSION}.zip"
curl -L "$REPO_URL" -o /tmp/repo.zip
unzip /tmp/repo.zip -d "$PACKAGE_DIR/$CLONE_DIR"
rm /tmp/repo.zip 

echo "Extracting dependencies from go.mod..."

awk '
/^require \($/,/^\)$/ {
    if ($1 != "require" && $1 != ")" && NF >= 2) {
        print $1, $2
    }
}
/^require [^(]/ {
    if (NF >= 3) {
        print $2, $3
    }
}
' go.mod | while read -r mod ver; do
    [[ -z "$mod" || -z "$ver" ]] && continue
    
    echo "Processing: $mod@$ver"
    
    # Download this specific module
    echo "  Downloading..."
    if ! go mod download "$mod@$ver" 2>/dev/null; then
        echo "  Error: Failed to download $mod@$ver"
        continue
    fi
    
    # Get the module path from cache
    modpath=$(go list -f '{{.Dir}}' -m "$mod@$ver" 2>/dev/null)
    
    # Copy module to target directory
    if [ -d "$modpath" ]; then
        echo "  Copying from: $modpath"
        cp -r --no-preserve=mode,ownership "$modpath" "$TARGET_DIR/"
    else
        echo "  Warning: Module directory not found for $mod@$ver"
    fi
done

echo "All sources and repo archive are in $PACKAGE_DIR/"
mv "$TARGET_DIR" $PACKAGE_DIR

cd $PACKAGE_DIR
echo "Creating package archive..."
tar -czf "../orch-cli-package.tar.gz" \
    "orch-cli" \
    "$CLONE_DIR" \
    "$(basename "$TARGET_DIR")"

cd ..

echo "Package created at .$PACKAGE_DIR/orch-cli-package.tar.gz"
