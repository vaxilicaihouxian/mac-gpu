#!/bin/bash
echo "编译 mac_gpu..."
go build -o mac_gpu main.go
echo "编译完成！"
echo ""
echo "运行程序需要管理员权限，请使用:"
echo "  sudo ./mac_gpu"
