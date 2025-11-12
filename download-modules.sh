#!/bin/bash
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

OUTPUT_DIR="dependencies"
OWN_MODULE="github.com/open-edge-platform/cli"
LICENSE_REPORT="dependencies-licenses.txt"
LICENSE_TEMP=$(mktemp)

# Common license file names to search for
LICENSE_FILES=(
    "LICENSE"
    "LICENSE.txt"
    "LICENSE.md"
    "COPYING"
    "COPYING.txt"
    "COPYING.md"
    "LICENCE"
    "LICENCE.txt"
    "LICENCE.md"
    "LICENSE-APACHE"
    "LICENSE-MIT"
    "LICENSE-BSD"
    "UNLICENSE"
)

# Error handler
error_exit() {
    echo "ERROR: $1" >&2
    exit 1
}

# Check prerequisites
command -v go >/dev/null 2>&1 || error_exit "go command not found. Please install Go."
[ -f "go.mod" ] || error_exit "go.mod not found in current directory."

echo "Parsing go.mod and downloading dependencies..."
mkdir -p "$OUTPUT_DIR" || error_exit "Failed to create output directory: $OUTPUT_DIR"

TOTAL=0
SUCCESS=0
FAILED=0
LICENSE_FOUND=0
LICENSE_NOT_FOUND=0

# Function to detect license type from file content
detect_license_type() {
    local file="$1"
    
    if [ ! -f "$file" ]; then
        echo "UNKNOWN"
        return
    fi
    
    local content=$(cat "$file" 2>/dev/null | tr '[:upper:]' '[:lower:]' || echo "")
    
    if [ -z "$content" ]; then
        echo "UNKNOWN"
        return
    fi
    
    # MPL - Check FIRST before GPL (MPL text mentions GPL in compatibility section)
    if echo "$content" | grep -qi "mozilla public license"; then
        if echo "$content" | grep -qi "version 2.0"; then
            echo "MPL-2.0"
            return
        fi
        echo "MPL"
        return
    fi
    
    # Apache
    if echo "$content" | grep -qi "apache license"; then
        if echo "$content" | grep -qi "version 2.0"; then
            echo "Apache-2.0"
            return
        fi
        echo "Apache"
        return
    fi
    
    # MIT
    if echo "$content" | grep -qi "mit license" || \
       (echo "$content" | grep -qi "permission is hereby granted" && \
        echo "$content" | grep -qi "without restriction"); then
        echo "MIT"
        return
    fi
    
    # BSD - More specific detection for 2-clause vs 3-clause
    if echo "$content" | grep -qi "bsd license" || \
       (echo "$content" | grep -qi "redistribution and use" && \
        ! echo "$content" | grep -qi "gnu general public license"); then
        
        # Check for explicit "3-clause" or "three-clause" mention
        if echo "$content" | grep -Eqi "(3-clause|three-clause)"; then
            echo "BSD-3-Clause"
            return
        fi
        
        # Check for explicit "2-clause" or "two-clause" mention
        if echo "$content" | grep -Eqi "(2-clause|two-clause)"; then
            echo "BSD-2-Clause"
            return
        fi
        
        # Count the number of conditions/clauses in the license
        # BSD-3-Clause has "neither the name" clause, BSD-2-Clause doesn't
        if echo "$content" | grep -qi "neither the name"; then
            echo "BSD-3-Clause"
            return
        fi
        
        # BSD-2-Clause typically has only two numbered conditions
        # Look for specific BSD-2-Clause patterns
        if echo "$content" | grep -qi "redistributions of source code must retain" && \
           echo "$content" | grep -qi "redistributions in binary form must reproduce" && \
           ! echo "$content" | grep -qi "neither the name" && \
           ! echo "$content" | grep -qi "endorse.*promote"; then
            echo "BSD-2-Clause"
            return
        fi
        
        # BSD-3-Clause has endorsement/promotion restriction
        if echo "$content" | grep -qi "redistributions of source code must retain" && \
           echo "$content" | grep -qi "redistributions in binary form must reproduce" && \
           (echo "$content" | grep -qi "neither the name" || \
            echo "$content" | grep -Eqi "(endorse|promote)"); then
            echo "BSD-3-Clause"
            return
        fi
        
        # Default to BSD if we can't determine the variant
        echo "BSD"
        return
    fi
    
    # ISC (check before GPL)
    if echo "$content" | grep -qi "isc license" || \
       (echo "$content" | grep -qi "permission to use, copy, modify" && \
        echo "$content" | grep -qi "isc"); then
        echo "ISC"
        return
    fi
    
    # Unlicense (check before GPL)
    if echo "$content" | grep -qi "this is free and unencumbered software"; then
        echo "Unlicense"
        return
    fi
    
    # GPL - Check after MPL, BSD, ISC
    if echo "$content" | grep -qi "gnu general public license"; then
        if echo "$content" | grep -qi "version 3"; then
            echo "GPL-3.0"
            return
        fi
        if echo "$content" | grep -qi "version 2"; then
            echo "GPL-2.0"
            return
        fi
        echo "GPL"
        return
    fi
    
    # LGPL
    if echo "$content" | grep -qi "gnu lesser general public license"; then
        if echo "$content" | grep -qi "version 3"; then
            echo "LGPL-3.0"
            return
        fi
        if echo "$content" | grep -qi "version 2"; then
            echo "LGPL-2.0"
            return
        fi
        echo "LGPL"
        return
    fi
    
    # If copyright found but unknown license
    if echo "$content" | grep -qi "copyright"; then
        echo "UNKNOWN_COPYRIGHT"
        return
    fi
    
    echo "UNKNOWN"
}

# Function to find and identify license
find_license() {
    local mod_dir="$1"
    local mod_name="$2"
    
    # Try to find license file (case-insensitive)
    for license_name in "${LICENSE_FILES[@]}"; do
        # Check exact match
        if [ -f "$mod_dir/$license_name" ]; then
            local license_type=$(detect_license_type "$mod_dir/$license_name")
            echo "$license_type|$mod_name ($license_name)"
            return 0
        fi
        
        # Check lowercase
        local lowercase=$(echo "$license_name" | tr '[:upper:]' '[:lower:]')
        if [ -f "$mod_dir/$lowercase" ]; then
            local license_type=$(detect_license_type "$mod_dir/$lowercase")
            echo "$license_type|$mod_name ($lowercase)"
            return 0
        fi
        
        # Check uppercase
        local uppercase=$(echo "$license_name" | tr '[:lower:]' '[:upper:]')
        if [ -f "$mod_dir/$uppercase" ]; then
            local license_type=$(detect_license_type "$mod_dir/$uppercase")
            echo "$license_type|$mod_name ($uppercase)"
            return 0
        fi
    done
    
    # Try to find any file with "license" in the name
    local found=$(find "$mod_dir" -maxdepth 1 -type f \( -iname "*license*" -o -iname "*licence*" \) 2>/dev/null | head -1 || true)
    if [ -n "$found" ]; then
        local license_type=$(detect_license_type "$found")
        local filename=$(basename "$found")
        echo "$license_type|$mod_name ($filename)"
        return 0
    fi
    
    # No license file found
    echo "NOT_FOUND|$mod_name"
    return 1
}

# Create temporary file to store module list
TEMP_MODULES=$(mktemp)
trap "rm -f $TEMP_MODULES $LICENSE_TEMP" EXIT

echo "Extracting module list from go.mod..."

# Parse go.mod directly to extract only modules listed in require blocks
awk '
    /^require \(/ { in_block=1; next }
    /^\)/ { in_block=0; next }
    in_block && /^[[:space:]]+[a-z]/ {
        gsub(/^[[:space:]]+/, "")
        gsub(/[[:space:]]*\/\/.*$/, "")
        if ($1 != "" && $2 != "") print $1, $2
    }
    /^require [a-z]/ && !/\(/ {
        gsub(/require[[:space:]]+/, "")
        gsub(/[[:space:]]*\/\/.*$/, "")
        if ($1 != "" && $2 != "") print $1, $2
    }
' go.mod > "$TEMP_MODULES" || error_exit "Failed to parse go.mod"

module_count=$(wc -l < "$TEMP_MODULES")
echo "Found $module_count modules in go.mod"
echo ""

# Process each module
while read -r path version; do
    # Skip empty lines or our own module
    if [ -z "$path" ] || [ "$path" = "$OWN_MODULE" ]; then
        continue
    fi
    
    TOTAL=$((TOTAL + 1))
    
    echo "[$TOTAL/$module_count] Processing $path@$version..."
    
    # Use go mod download to download to cache
    if go mod download "$path@$version" 2>&1; then
        # Get the module directory from cache
        mod_dir=$(go list -m -f '{{.Dir}}' "$path@$version" 2>/dev/null || echo "")
        
        if [ -n "$mod_dir" ] && [ -d "$mod_dir" ]; then
            # Create safe directory name
            safe_name=$(echo "${path}@${version}" | tr '/' '_')
            dest_dir="$OUTPUT_DIR/$safe_name"
            
            # Copy module from cache
            if cp -r "$mod_dir" "$dest_dir" 2>/dev/null; then
                SUCCESS=$((SUCCESS + 1))
                echo "  ✓ Copied to $dest_dir"
                
                # Find and identify license
                license_info=$(find_license "$dest_dir" "$safe_name" || echo "NOT_FOUND|$safe_name")
                if echo "$license_info" | grep -q "^NOT_FOUND|"; then
                    LICENSE_NOT_FOUND=$((LICENSE_NOT_FOUND + 1))
                    echo "  ⚠ License not found"
                else
                    LICENSE_FOUND=$((LICENSE_FOUND + 1))
                    license_only=$(echo "$license_info" | cut -d'|' -f1)
                    echo "  ✓ License: $license_only"
                fi
                
                # Add to temporary file (format: LICENSE_TYPE|MODULE_INFO)
                echo "$license_info" >> "$LICENSE_TEMP"
            else
                FAILED=$((FAILED + 1))
                echo "  ✗ Failed to copy from $mod_dir to $dest_dir"
            fi
        else
            FAILED=$((FAILED + 1))
            echo "  ✗ Module directory not found (expected: $mod_dir)"
        fi
    else
        FAILED=$((FAILED + 1))
        echo "  ✗ Download failed for $path@$version"
    fi
done < "$TEMP_MODULES"

echo ""
echo "Generating sorted license report..."

# Create license report header
cat > "$LICENSE_REPORT" <<EOF
# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
#
# Go Module Dependencies License Report
# Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
# Sorted by license type
#

EOF

# Sort by license type (first field) and append to report
# Format in temp file: LICENSE_TYPE|MODULE_NAME (file)
# Output format: MODULE_NAME (file) LICENSE_TYPE
sort -t'|' -k1,1 "$LICENSE_TEMP" | while IFS='|' read -r license_type module_info; do
    echo "$module_info $license_type" >> "$LICENSE_REPORT"
done

echo ""
echo "================================"
echo "Download Summary"
echo "================================"
echo "Total modules in go.mod: $TOTAL"
echo "Successfully downloaded: $SUCCESS"
echo "Failed: $FAILED"
echo ""
echo "License Summary"
echo "================================"
echo "Licenses found: $LICENSE_FOUND"
echo "Licenses not found: $LICENSE_NOT_FOUND"
echo ""

# Display license type distribution
echo "License Type Distribution:"
echo "================================"
sort -t'|' -k1,1 "$LICENSE_TEMP" | cut -d'|' -f1 | uniq -c | sort -rn | while read count license; do
    printf "  %-25s %3d\n" "$license" "$count"
done

echo ""
echo "Output directory: $OUTPUT_DIR/"
echo "License report: $LICENSE_REPORT"
echo ""

if [ $FAILED -gt 0 ]; then
    echo "⚠ Some modules failed to download. Review the output above."
    exit 1
fi

echo "✓ All dependencies from go.mod downloaded and scanned successfully!"
