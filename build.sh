#!/bin/bash

MAKE_PACKAGE=false

usage() {
 echo "Usage: $0 [OPTIONS]"
 echo "Options:"
 echo " --package      Make a delivery package instead of plain binary"
}

# Function to handle options and arguments
handle_options() {
  while [ $# -gt 0 ]; do
    case $1 in
      --package)
        MAKE_PACKAGE=true
        ;;
      *)
        echo "Invalid option: $1" >&2
        usage
        exit 1
        ;;
    esac
    shift
  done
}

# Main script execution
handle_options "$@"

if [ ! -d "./build" ]; then
  mkdir ./build
fi

## SERVER

platforms=("linux/amd64" "linux/arm64" "linux/riscv64" "linux/ppc64le")
CGO_ENABLED=0

for platform in "${platforms[@]}"; do
    echo "* Compiling server for $platform..."
    platform_split=(${platform//\// })

    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}

    go build -o build/server_${platform_split[0]}_${platform_split[1]}.bin ./cmd/server

    if [ "$MAKE_PACKAGE" == "true" ]; then
        tar -czf build/server_${platform_split[0]}_${platform_split[1]}.tar.gz build/server_${platform_split[0]}_${platform_split[1]}.bin
        rm build/server_${platform_split[0]}_${platform_split[1]}.bin
    fi
done

## CLIENT

platforms=("windows/amd64" "windows/arm64" "darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64")

for platform in "${platforms[@]}"; do
    echo "* Compiling client for $platform..."
    platform_split=(${platform//\// })

    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}

    go build -o build/cli_${platform_split[0]}_${platform_split[1]}.bin ./cmd/cli
    if [ "$MAKE_PACKAGE" == "true" ]; then
        tar -czf build/cli_${platform_split[0]}_${platform_split[1]}.tar.gz build/cli_${platform_split[0]}_${platform_split[1]}.bin
        rm build/cli_${platform_split[0]}_${platform_split[1]}.bin
    fi
done
