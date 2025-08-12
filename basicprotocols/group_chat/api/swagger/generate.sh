#!/bin/bash

# 群聊模块 Swagger 文档生成脚本

set -e

echo "🚀 开始生成群聊模块的 Swagger 文档..."

# 检查 swag 工具是否安装
if ! command -v swag &> /dev/null; then
    echo "❌ swag 工具未安装，正在安装..."
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# 创建 docs 目录（如果不存在）
mkdir -p docs

# 生成 swagger 文档
echo "📝 生成 Swagger 文档..."
swag init \
    -g ../../cmd/main.go \
    -o ./docs \
    --parseDependency \
    --parseInternal \
    --parseDepth 3 \
    --parseVendor \
    --generatedTime \
    --instanceName group_chat_swagger

# 检查生成是否成功
if [ -f "docs/group_chat_swagger_docs.go" ]; then
    echo "✅ Swagger 文档生成成功！"
    echo "📁 生成的文件："
    ls -la docs/
    echo ""
    echo "🌐 访问地址：http://0.0.0.0:7568/group-chat/docs/index.html"
    echo "📖 文档文件位置：./docs/"
    echo ""
    echo "⚠️  重要提醒："
    echo "   如果添加了新的 API 接口，请手动更新 docs/group_chat_swagger_swagger.json 文件"
    echo "   确保新接口的路径定义包含在 paths 部分中"
else
    echo "❌ Swagger 文档生成失败！"
    exit 1
fi

echo "🎉 完成！" 