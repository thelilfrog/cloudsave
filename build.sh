#!/bin/bash

MAKE_PACKAGE=false
VERSION=0.0.4

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

for platform in "${platforms[@]}"; do
    echo "* Compiling server for $platform..."
    platform_split=(${platform//\// })

    EXT=""
    if [ "${platform_split[0]}" == "windows" ]; then
      EXT=.exe
    fi

    if [ "$MAKE_PACKAGE" == "true" ]; then
        CGO_ENABLED=0 GOOS=${platform_split[0]} GOARCH=${platform_split[1]} GORISCV64=rva22u64 GOAMD64=v3 GOARM64=v8.2 go build -o build/cloudsave_server$EXT -a ./cmd/server
        tar -czf build/server_${platform_split[0]}_${platform_split[1]}.tar.gz build/cloudsave_server$EXT
        rm build/cloudsave_server$EXT
    else
      CGO_ENABLED=0 GOOS=${platform_split[0]} GOARCH=${platform_split[1]} GORISCV64=rva22u64 GOAMD64=v3 GOARM64=v8.2 go build -o build/cloudsave_server_${platform_split[0]}_${platform_split[1]}$EXT -a ./cmd/server
    fi
done

# WEB

platforms=("linux/amd64" "linux/arm64" "linux/riscv64" "linux/ppc64le")

for platform in "${platforms[@]}"; do
    echo "* Compiling web server for $platform..."
    platform_split=(${platform//\// })

    EXT=""
    if [ "${platform_split[0]}" == "windows" ]; then
      EXT=.exe
    fi

    if [ "$MAKE_PACKAGE" == "true" ]; then
        CGO_ENABLED=0 GOOS=${platform_split[0]} GOARCH=${platform_split[1]} go build -o build/cloudsave_web$EXT -a ./cmd/web
        tar -czf build/web_${platform_split[0]}_${platform_split[1]}.tar.gz build/cloudsave_web$EXT
        rm build/cloudsave_web$EXT
    else
      CGO_ENABLED=0 GOOS=${platform_split[0]} GOARCH=${platform_split[1]} go build -o build/cloudsave_web_${platform_split[0]}_${platform_split[1]}$EXT -a ./cmd/web
    fi
done

## CLIENT

platforms=("windows/amd64" "windows/arm64" "darwin/amd64" "darwin/arm64" "linux/amd64" "linux/arm64")

for platform in "${platforms[@]}"; do
    echo "* Compiling client for $platform..."
    platform_split=(${platform//\// })

    EXT=""
    if [ "${platform_split[0]}" == "windows" ]; then
      EXT=.exe
    fi

    if [ "$MAKE_PACKAGE" == "true" ]; then
        CGO_ENABLED=0 GOOS=${platform_split[0]} GOARCH=${platform_split[1]} go build -o build/cloudsave$EXT -a ./cmd/cli
        tar -czf build/cli_${platform_split[0]}_${platform_split[1]}.tar.gz build/cloudsave$EXT
        rm build/cloudsave$EXT
    else
        CGO_ENABLED=0 GOOS=${platform_split[0]} GOARCH=${platform_split[1]} go build -o build/cloudsave_${platform_split[0]}_${platform_split[1]}$EXT -a ./cmd/cli
    fi
done
