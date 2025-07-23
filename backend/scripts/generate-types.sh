#!/bin/bash

# Script to generate TypeScript types from Go structs

echo "Generating TypeScript types from Go structs..."

# Run tygo to generate types
~/go/bin/tygo generate

if [ $? -eq 0 ]; then
    echo "TypeScript types generated successfully at api-types.ts"
else
    echo "Failed to generate TypeScript types"
    exit 1
fi