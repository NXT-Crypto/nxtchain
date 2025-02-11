#!/bin/bash

# Prompt user for build target name
echo "Enter the name of the build target (node, miner, or wallet):"
read BUILD_NAME

# Validate input
if [[ "$BUILD_NAME" != "node" && "$BUILD_NAME" != "miner" && "$BUILD_NAME" != "wallet" ]]; then
    echo "Invalid build name. Please enter 'node', 'miner', or 'wallet'."
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

# Navigate to the subdirectory of the build target
if [ -d "$BUILD_NAME" ]; then
    cd "$BUILD_NAME" || exit
else
    echo "Error: Directory $BUILD_NAME does not exist."
    exit 1
fi

# Loop through OS and architectures
for OS in "${!OS_ARCH[@]}"; do
    for ARCH in ${OS_ARCH[$OS]}; do
        # Create subdirectories
        BUILD_DIR="../$OUTPUT_DIR/$BUILD_NAME/$OS"
        mkdir -p "$BUILD_DIR"
        
        # Define output file name
        OUTPUT_FILE="$BUILD_DIR/${BUILD_NAME}_${OS}_${ARCH}"
        
        # Add .exe extension for Windows builds
        if [ "$OS" == "windows" ]; then
            OUTPUT_FILE+=".exe"
        fi
        
        # Build command
        echo "Building $BUILD_NAME for $OS-$ARCH..."
        env GOOS=$OS GOARCH=$ARCH go build -o "$OUTPUT_FILE" .
        
        if [ $? -eq 0 ]; then
            echo "Build successful: $OUTPUT_FILE"
        else
            echo "Build failed for $OS-$ARCH"
        fi
    done
done

echo "All builds completed."

# Return to original directory
cd -
