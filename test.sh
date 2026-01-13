#!/bin/bash

echo "测试 mac_gpu 程序"
echo "=================="
echo ""

# 测试1: 检查编译是否成功
echo "测试1: 检查可执行文件"
if [ -f "mac_gpu" ]; then
    echo "✓ mac_gpu 可执行文件存在"
    ls -lh mac_gpu
else
    echo "✗ mac_gpu 可执行文件不存在"
    exit 1
fi
echo ""

# 测试2: 运行程序（不使用sudo，应该显示警告）
echo "测试2: 运行程序（无sudo）"
timeout 3 ./mac_gpu < /dev/null || true
echo ""

# 测试3: 检查是否检测到M2芯片
echo "测试3: 验证GPU信息检测"
output=$(timeout 3 ./mac_gpu < /dev/null 2>&1 || true)
if echo "$output" | grep -q "Apple M"; then
    echo "✓ 成功检测到芯片型号"
    echo "$output" | grep "芯片型号" || echo "$output" | grep "检测到芯片"
else
    echo "✗ 未能检测到芯片型号"
fi
echo ""

echo "测试完成！"
echo ""
echo "要查看完整的GPU监控信息，请使用: sudo ./mac_gpu"
