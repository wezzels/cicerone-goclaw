#!/bin/bash
# Build papers from LaTeX sources
# Usage: ./scripts/build-papers.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PAPER_DIR="${SCRIPT_DIR}/../docs/paper"
OUTPUT_DIR="${SCRIPT_DIR}/../docs/paper/output"

echo "=== Building Cicerone Papers ==="
echo "Paper directory: $PAPER_DIR"
echo "Output directory: $OUTPUT_DIR"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Check for pdflatex
if ! command -v pdflatex &> /dev/null; then
    echo "pdflatex not found - skipping LaTeX compilation"
    echo "Papers will be built when LaTeX is available"
    exit 0
fi

# Function to build a paper
build_paper() {
    local tex_file="$1"
    local base_name=$(basename "$tex_file" .tex)
    
    echo ""
    echo "Building: $base_name"
    echo "Source: $tex_file"
    
    cd "$PAPER_DIR"
    
    # First pass
    pdflatex -interaction=nonstopmode -output-directory="$OUTPUT_DIR" "$tex_file" || true
    
    # Second pass for references
    pdflatex -interaction=nonstopmode -output-directory="$OUTPUT_DIR" "$tex_file" || true
    
    if [ -f "$OUTPUT_DIR/${base_name}.pdf" ]; then
        echo "✓ Built: $OUTPUT_DIR/${base_name}.pdf"
    else
        echo "✗ Failed: $base_name"
    fi
}

# Build all papers
for tex_file in "$PAPER_DIR"/*.tex; do
    if [ -f "$tex_file" ]; then
        build_paper "$tex_file"
    fi
done

echo ""
echo "=== Build Complete ==="
echo "Output directory: $OUTPUT_DIR"
ls -la "$OUTPUT_DIR"/*.pdf 2>/dev/null || echo "No PDFs generated"

exit 0
