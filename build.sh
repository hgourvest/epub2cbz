#!/bin/bash
set -e

# Name of your binary
BINARY_NAME="epub2cbz"

# Output directory
BUILD_DIR="build"

# Create build directory
mkdir -p $BUILD_DIR

# List of platforms (GOOS/GOARCH)
PLATFORMS="linux/amd64 linux/386 linux/arm64
           darwin/amd64 darwin/arm64
           windows/amd64 windows/386"

echo "Starting cross-platform builds..."

for platform in $PLATFORMS; do
    GOOS=$(echo $platform | cut -d'/' -f1)
    GOARCH=$(echo $platform | cut -d'/' -f2)
    
    echo "Building for $GOOS/$GOARCH..."
    
    # Set environment variables
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    
    # Binary name according to platform
    if [ $GOOS = "windows" ]; then
        BINARY_NAME_PLATFORM="${BINARY_NAME}_${GOOS}_${GOARCH}.exe"
    else
        BINARY_NAME_PLATFORM="${BINARY_NAME}_${GOOS}_${GOARCH}"
    fi
    
    # Build
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build -o "$BUILD_DIR/$BINARY_NAME_PLATFORM" .
    
    # Reset variables
    unset GOOS
    unset GOARCH
    
    echo "Build completed: $BUILD_DIR/$BINARY_NAME_PLATFORM"
done

echo "All cross-platform builds are completed!"