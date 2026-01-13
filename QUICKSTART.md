# Mac GPU Monitor - 快速入门

## 一键运行

### 首次使用
```bash
# 1. 编译程序
go build -o mac_gpu main.go

# 2. 运行（需要sudo权限）
sudo ./mac_gpu
```

### 使用脚本
```bash
# 编译
./build.sh

# 测试
./test.sh
```

## 核心功能

✓ 自动检测 Apple M1/M2/M3 芯片
✓ 显示 GPU 核心数量
✓ 实时 GPU 使用率（进度条）
✓ 显存使用情况（当前/最大）
✓ 最近60秒历史图表

## 常见命令

```bash
# 编译
go build -o mac_gpu main.go

# 运行
sudo ./mac_gpu

# 退出
Ctrl+C
```

## 重要提示

⚠️ **必须使用 sudo 运行程序**
```bash
sudo ./mac_gpu
```

原因：程序使用 `powermetrics` 命令获取 GPU 信息，该命令需要管理员权限。

## 文件说明

- `main.go` - 主程序源代码
- `go.mod` - Go 模块配置
- `mac_gpu` - 编译后的可执行文件
- `build.sh` - 构建脚本
- `test.sh` - 测试脚本
- `README.md` - 项目说明
- `USAGE.md` - 详细使用说明
- `QUICKSTART.md` - 本文件

## 系统要求

- macOS 11.0+
- Apple Silicon Mac (M1/M2/M3)
- Go 1.22+ (编译需要)
- 管理员权限 (运行需要)

## 问题排查

**问题**: 无法获取 GPU 信息
**解决**: 使用 sudo 运行程序

**问题**: 编译失败
**解决**: 检查 Go 版本 `go version`

## 更多信息

详细使用说明请查看: [USAGE.md](USAGE.md)
