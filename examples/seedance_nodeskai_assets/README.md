# Seedance NoDesk AI Assets Example

这个目录是一个最小联调示例，只做两步：

1. 上传当前目录的 [`test.png`](./test.png)
2. 从上传响应里取出 `asset_id`，立刻查询一次素材详情

## 用法

先设置 OAuth2 凭证：

```bash
export NODESKAI_CLIENT_ID=ndapp_xxx
export NODESKAI_CLIENT_SECRET=your-secret
```

然后执行：

```bash
./main.sh <group_id> [name]
```

示例：

```bash
./main.sh grp_abc123
./main.sh grp_abc123 "女性正脸-01"
```

## 文件

- [`main.sh`](./main.sh): 上传 `test.png` 后立即查询详情，并打印完整过程日志
- [`test.png`](./test.png): 默认上传测试图片
- [`main.go`](./main.go): `seedance_assets` 的 Go 版本示例，包含更完整的步骤日志、耗时和输入摘要
