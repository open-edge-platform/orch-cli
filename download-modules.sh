#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -e

OUTPUT_DIR="dependencies"
OWN_MODULE="github.com/open-edge-platform/cli"

echo "Parsing go.mod and downloading dependencies..."
mkdir -p "$OUTPUT_DIR"

TOTAL=0
SUCCESS=0
FAILED=0

# Parse go.mod directly to extract only modules listed in require blocks
awk '
    /^require \(/ { in_block=1; next }
    /^\)/ { in_block=0; next }
    in_block && /^[[:space:]]+[a-z]/ {
        # Extract module path and version
        gsub(/^[[:space:]]+/, "")
        gsub(/[[:space:]]*\/\/.*$/, "")
        print $1, $2
    }
    /^require [a-z]/ && !/\(/ {
        # Single-line require
        gsub(/require[[:space:]]+/, "")
        gsub(/[[:space:]]*\/\/.*$/, "")
        print $1, $2
    }
' go.mod | while read -r path version; do
    # Skip empty lines or our own module
    if [ -z "$path" ] || [ "$path" = "$OWN_MODULE" ]; then
        continue
    fi
    
    TOTAL=$((TOTAL + 1))
    
    echo "[$TOTAL] Downloading $path@$version..."
    
    # Use go mod download to download to cache
    if go mod download "$path@$version" >/dev/null 2>&1; then
        # Get the module directory from cache
        mod_dir=$(go list -m -f '{{.Dir}}' "$path@$version" 2>/dev/null)
        
        if [ -n "$mod_dir" ] && [ -d "$mod_dir" ]; then
            # Create safe directory name
            safe_name=$(echo "${path}@${version}" | tr '/' '_')
            dest_dir="$OUTPUT_DIR/$safe_name"
            
            # Copy module from cache
            if cp -r "$mod_dir" "$dest_dir" 2>/dev/null; then
                SUCCESS=$((SUCCESS + 1))
                echo "  ✓ Copied to $dest_dir"
            else
                FAILED=$((FAILED + 1))
                echo "  ✗ Failed to copy"
            fi
        else
            FAILED=$((FAILED + 1))
            echo "  ✗ Module directory not found"
        fi
    else
        FAILED=$((FAILED + 1))
        echo "  ✗ Download failed"
    fi
done

echo ""
echo "================================"
echo "Download Summary"
echo "================================"
echo "Total modules in go.mod: $TOTAL"
echo "Successfully downloaded: $SUCCESS"
echo "Failed: $FAILED"
echo "Output directory: $OUTPUT_DIR/"
echo ""

if [ $FAILED -gt 0 ]; then
    echo "⚠ Some modules failed to download. Check the output above."
    exit 1
fi

echo "✓ All dependencies from go.mod downloaded successfully!"
