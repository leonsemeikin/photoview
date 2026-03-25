#!/bin/bash

# Administrative script for Photoview

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_DIR="$(dirname "$SCRIPT_DIR")"

cd "$API_DIR" || exit 1

case "$1" in
	clean-users)
		go run admin/main.go clean-users
		;;
	clean-albums)
		go run admin/main.go clean-albums
		;;
	clean-path)
		go run admin/main.go clean-path "$2"
		;;
	*)
		echo "Usage: $0 {clean-users|clean-albums|clean-path}"
		exit 1
		;;
esac
