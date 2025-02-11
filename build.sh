#!/bin/bash

# Check if an argument was provided
if [ -z "$1" ]; then
    # Prompt user for build target name
    echo "Enter the name of the build target (node, miner, wallet, or all):"
    read BUILD_NAME
else
    BUILD_NAME="$1"
fi

# Validate input
if [[ "$BUILD_NAME" != "node" && "$BUILD_NAME" != "miner" && "$BUILD_NAME" != "wallet" && "$BUILD_NAME" != "all" ]]; then
    echo "Invalid build name. Please enter 'node', 'miner', 'wallet', or 'all'."
    exit 1
fi

# Define output directory
OUTPUT_DIR="build"

# Define target OS and architectures
declare -A OS_ARCH
OS_ARCH=(
    [darwin]="amd64 arm64"
    [linux]="386 amd64 arm arm64"
    [windows]="386 amd64"
)

# Create output directory if not exists
mkdir -p "$OUTPUT_DIR"

# Define build targets
if [ "$BUILD_NAME" == "all" ]; then
    BUILD_TARGETS=("node" "miner" "wallet")
else
    BUILD_TARGETS=("$BUILD_NAME")
fi

# Loop through each build target
for TARGET in "${BUILD_TARGETS[@]}"; do
    # Navigate to the subdirectory of the build target
    if [ -d "$TARGET" ]; then
        cd "$TARGET" || exit
    else
        echo "Error: Directory $TARGET does not exist."
        exit 1
    fi

    # Loop through OS and architectures
    for OS in "${!OS_ARCH[@]}"; do
        for ARCH in ${OS_ARCH[$OS]}; do
            # Create subdirectories
            BUILD_DIR="../$OUTPUT_DIR/${TARGET}/$OS"
            mkdir -p "$BUILD_DIR"
            
            # Define output file name
            OUTPUT_FILE="$BUILD_DIR/nxtchain_${TARGET}_${OS}_${ARCH}"
            
            # Add .exe extension for Windows builds
            if [ "$OS" == "windows" ]; then
                OUTPUT_FILE+=".exe"
            fi
            
            # Build command
            echo "Building $TARGET for $OS-$ARCH..."
            env GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT_FILE" .
            
            if [ $? -eq 0 ]; then
                echo "Build successful: $OUTPUT_FILE"
            else
                echo "Build failed for $OS-$ARCH"
            fi
        done
    done

    # Return to root directory
    cd -
done

echo "All builds completed."
