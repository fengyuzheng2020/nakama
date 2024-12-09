#!/bin/bash

# 项目名称
PROJECT_NAME="nakama"
# 源代码文件
SOURCE_FILE="main.go"

# 构建输出目录
OUTPUT_DIR="build"

# 支持的目标平台
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")

# 创建输出目录
mkdir -p $OUTPUT_DIR

# 遍历每个平台并构建
for PLATFORM in "${PLATFORMS[@]}"
do
    IFS="/" read -r -a SPLIT <<< "$PLATFORM"
    GOOS=${SPLIT[0]}
    GOARCH=${SPLIT[1]}

    OUTPUT_NAME="${PROJECT_NAME}-${GOOS}-${GOARCH}"

    # Windows 可执行文件需要 .exe 后缀
    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME+=".exe"
    fi

    echo "构建 $PROJECT_NAME for $GOOS/$GOARCH"

    env GOOS=$GOOS GOARCH=$GOARCH go build -o "$OUTPUT_DIR/$OUTPUT_NAME" $SOURCE_FILE

    if [ $? -ne 0 ]; then
        echo "构建 $PROJECT_NAME for $GOOS/$GOARCH 失败"
        exit 1
    fi
done

echo "所有平台构建完成，输出文件位于 $OUTPUT_DIR/"
