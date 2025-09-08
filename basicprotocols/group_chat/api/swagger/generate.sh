#!/bin/bash

# Group Chat Module Swagger Documentation Generation Script

set -e

echo "🚀 Starting to generate Swagger documentation for group chat module..."

# Check if swag tool is installed
if ! command -v swag &> /dev/null; then
    echo "❌ swag tool not installed, installing..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# Create docs directory (if it doesn't exist)
mkdir -p docs

# Generate swagger documentation
echo "📝 Generating Swagger documentation..."
swag init \
    -g ../../cmd/main.go \
    -o ./docs \
    --parseDependency \
    --parseInternal \
    --parseDepth 3 \
    --parseVendor \
    --generatedTime \
    --instanceName group_chat_swagger

# Check if generation was successful
if [ -f "docs/group_chat_swagger_docs.go" ]; then
    echo "✅ Swagger documentation generated successfully!"
    echo "📁 Generated files:"
    ls -la docs/
    echo ""
    echo "🌐 Access URL: http://0.0.0.0:7568/group-chat/docs/index.html"
    echo "📖 Documentation location: ./docs/"
    echo ""
    echo "⚠️  Important reminder:"
    echo "   If you added new API interfaces, please manually update the docs/group_chat_swagger_swagger.json file"
    echo "   Ensure new interface path definitions are included in the paths section"
else
    echo "❌ Swagger documentation generation failed!"
    exit 1
fi

echo "🎉 Complete!" 