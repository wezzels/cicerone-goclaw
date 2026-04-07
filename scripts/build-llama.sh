#!/bin/bash
# Build llama.cpp from source with CPU optimizations
# Usage: ./build-llama.sh [cpu|cuda|native]

set -e

BUILD_TYPE="${1:-cpu}"
LLAMA_DIR="/opt/llama.cpp"
BUILD_DIR="$LLAMA_DIR/build"

echo "=== Building llama.cpp ==="
echo "Build type: $BUILD_TYPE"
echo ""

# Check for dependencies
echo "Checking dependencies..."
command -v cmake >/dev/null 2>&1 || { echo "Installing cmake..."; apt-get update && apt-get install -y cmake; }
command -v git >/dev/null 2>&1 || { echo "Installing git..."; apt-get update && apt-get install -y git; }
command -v gcc >/dev/null 2>&1 || { echo "Installing gcc..."; apt-get update && apt-get install -y build-essential; }

# Check CPU capabilities for optimization
check_cpu_features() {
    echo "Checking CPU capabilities..."
    CPU_FLAGS=$(cat /proc/cpuinfo | grep -m1 'flags' | cut -d: -f2)
    
    HAS_AVX512=$(echo "$CPU_FLAGS" | grep -q 'avx512f' && echo 'yes' || echo 'no')
    HAS_AVX2=$(echo "$CPU_FLAGS" | grep -q 'avx2' && echo 'yes' || echo 'no')
    HAS_AVX=$(echo "$CPU_FLAGS" | grep -q 'avx' && echo 'yes' || echo 'no')
    HAS_FMA=$(echo "$CPU_FLAGS" | grep -q 'fma' && echo 'yes' || echo 'no')
    
    echo "  AVX-512: $HAS_AVX512"
    echo "  AVX2: $HAS_AVX2"
    echo "  AVX: $HAS_AVX"
    echo "  FMA: $HAS_FMA"
    
    # Return recommended instruction set
    if [ "$HAS_AVX512" = "yes" ]; then
        echo "NATIVE"
    elif [ "$HAS_AVX2" = "yes" ]; then
        echo "AVX2"
    elif [ "$HAS_AVX" = "yes" ]; then
        echo "AVX"
    else
        echo "SSE"
    fi
}

# Check CUDA (optional)
if [ "$BUILD_TYPE" = "cuda" ]; then
    if ! command -v nvcc >/dev/null 2>&1; then
        echo "WARNING: CUDA not found, falling back to optimized CPU build"
        BUILD_TYPE="cpu"
    else
        echo "CUDA found: $(nvcc --version | head -1)"
    fi
fi

# Clone if not exists
if [ ! -d "$LLAMA_DIR" ]; then
    echo "Cloning llama.cpp..."
    git clone https://github.com/ggerganov/llama.cpp "$LLAMA_DIR"
fi

cd "$LLAMA_DIR"

# Pull latest
echo "Updating repository..."
git pull origin master || git pull origin main

# Create build directory
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Configure build with CPU optimizations
echo "Configuring build..."

CMAKE_OPTS="-DCMAKE_BUILD_TYPE=Release"

if [ "$BUILD_TYPE" = "cuda" ]; then
    echo "Building with CUDA support..."
    CMAKE_OPTS="$CMAKE_OPTS -DGGML_CUDA=ON"
elif [ "$BUILD_TYPE" = "native" ]; then
    echo "Building with native CPU optimizations..."
    CMAKE_OPTS="$CMAKE_OPTS -DGGML_NATIVE=ON"
else
    echo "Building with optimized CPU support..."
    # Enable all CPU optimizations
    CMAKE_OPTS="$CMAKE_OPTS -DGGML_NATIVE=ON"
    
    # Check CPU features and set appropriate flags
    CPU_LEVEL=$(check_cpu_features)
    echo "CPU optimization level: $CPU_LEVEL"
    
    case "$CPU_LEVEL" in
        NATIVE|AVX512)
            echo "Enabling AVX-512 optimizations..."
            CMAKE_OPTS="$CMAKE_OPTS -DGGML_AVX512=ON"
            ;;
        AVX2)
            echo "Enabling AVX2 optimizations..."
            CMAKE_OPTS="$CMAKE_OPTS -DGGML_AVX2=ON -DGGML_FMA=ON -DGGML_F16C=ON"
            ;;
        AVX)
            echo "Enabling AVX optimizations..."
            CMAKE_OPTS="$CMAKE_OPTS -DGGML_AVX=ON"
            ;;
    esac
fi

# Enable additional optimizations
CMAKE_OPTS="$CMAKE_OPTS -DGGML_BLAS=ON -DGGML_BLAS_VENDOR=OpenBLAS"

echo "CMake options: $CMAKE_OPTS"
cmake .. $CMAKE_OPTS

# Build with all cores
echo "Building with $(nproc) cores..."
make -j$(nproc)

# Verify build
echo ""
echo "Build outputs:"
ls -la bin/

# Test binary
if [ -f bin/llama-server ]; then
    echo ""
    echo "llama-server built successfully!"
    bin/llama-server --version || true
fi

echo ""
echo "=== llama.cpp Build Complete ==="
echo "Build type: $BUILD_TYPE"
echo "Binaries location: $BUILD_DIR/bin/"
echo "Server binary: $BUILD_DIR/bin/llama-server"
echo ""
echo "Recommended models for CPU inference:"
echo "  - gemma-2-2b-it (1.5 GB) - Fast, good for testing"
echo "  - phi-3-mini (2.3 GB) - Lightweight, efficient"
echo "  - tinyllama (0.6 GB) - Minimal resource usage"
echo ""
echo "Usage:"
echo "  llama-server --model /opt/models/model.gguf --port 8080 --ctx-size 4096 --threads \$(nproc)"

# Create symlink for easy access
ln -sf "$BUILD_DIR/bin/llama-server" /usr/local/bin/llama-server 2>/dev/null || true

exit 0