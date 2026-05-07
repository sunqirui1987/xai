# Seedance NoDesk AI Assets Example

这个目录是一个最小联调示例，支持三步：

1. 创建素材组
2. 上传当前目录的 [`test.png`](./test.png)
3. 从上传响应里取出 `asset_id`，立刻查询一次素材详情

## 用法

先设置 OAuth2 凭证：

```bash
export NODESKAI_CLIENT_ID=ndapp_xxx
export NODESKAI_CLIENT_SECRET=your-secret
export NODESKAI_EXTERNAL_USER_ID=your-app-user-id
```

然后执行：

```bash
go run ./main.go create-group "默认素材组" "Seedance 2.0 默认素材组"

./main.sh [group_name] [asset_name]
```

示例：

```bash
./main.sh
./main.sh "默认素材组"
./main.sh "人像素材-正式环境" "女性正脸-01"
```

`main.sh` 现在会固定先创建素材组，再上传测试图并查询详情。

## 文件

- [`main.sh`](./main.sh): 先创建素材组，再上传 `test.png` 并查询详情
- [`test.png`](./test.png): 默认上传测试图片
- [`main.go`](./main.go): `seedance_assets` 的 Go 版本示例，包含创建素材组、上传、查询和更完整的日志
