#!/bin/bash
# Setup script permissions for Linux/macOS

echo "Setting up executable permissions..."

chmod +x build.sh
chmod +x demo.sh
chmod +x test-e2e.sh
chmod +x deploy/install.sh

echo "Permissions set!"
echo
echo "Available scripts:"
echo "  ./build.sh     - Build binaries"
echo "  ./demo.sh      - Run demo"
echo "  ./test-e2e.sh  - Run end-to-end tests"
echo "  ./deploy/install.sh - Install system-wide (requires sudo)"
echo
echo "Or use Makefile targets:"
echo "  make help      - Show all available targets"
