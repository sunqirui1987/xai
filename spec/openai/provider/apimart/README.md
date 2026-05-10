# APIMart GPT-Image-2 Provider

`spec/openai/provider/apimart` 为 APIMart 的 `gpt-image-2` 提供 `xai` 兼容接入。

它基于 APIMart 的 OpenAI 风格图片接口实现，但和标准 `openai/gpt-image-2` 的一个关键差异是：

- 提交生成请求时返回的是 `task_id`
- 需要继续轮询 `GET /v1/tasks/{task_id}` 获取最终图片 URL

因此本包实现的是一个“异步图片 provider”。

## API Reference

- 图像生成: `POST /v1/images/generations`
- 任务查询: `GET /v1/tasks/{task_id}`
- 文档页:
  - `https://docs.apimart.ai/cn/api-reference/images/gpt-image-2/generation`
  - `https://docs.apimart.ai/cn/api-reference/tasks/status`

## 包内实现

- [service.go](./service.go)
- [image.go](./image.go)
- [service_test.go](./service_test.go)

## 支持能力

- `xai.GenImage`
- `xai.EditImage`
- `xai.GetTask(...)` 恢复异步任务
- `xai.Wait(...)` 自动轮询等待结果

目前仅支持 `gpt-image-2`。

可接受的模型名：

- `gpt-image-2`
- `apimart/gpt-image-2`

## 快速开始

### 方式一：直接创建 Service

```go
import (
    "os"

    apimart "github.com/goplus/xai/spec/openai/provider/apimart"
)

svc := apimart.NewService(os.Getenv("APIMART_API_KEY"))
```

支持运行时更新 API Key：

```go
svc.SetApiKey("new-api-key")
```

### 方式二：Register 到全局 xai

```go
import (
    "context"

    xai "github.com/goplus/xai/spec"
    apimart "github.com/goplus/xai/spec/openai/provider/apimart"
)

apimart.Register("your-api-key")

svc, err := xai.New(context.Background(), "apimart:")
if err != nil {
    panic(err)
}
_ = svc
```

也支持通过 URI 覆盖 `key` / `base`：

```go
svc, err := xai.New(ctx, "apimart:key=xxx&base=https%3A%2F%2Fapi.apimart.ai%2Fv1%2F")
```

## 配置项

```go
svc := apimart.NewService("your-api-key",
    apimart.WithBaseURL("https://api.apimart.ai/v1/"),
    apimart.WithHTTPClient(&http.Client{}),
    apimart.WithDebugLog(true),
)
```

可用配置：

- `apimart.WithBaseURL(...)`
- `apimart.WithHTTPClient(...)`
- `apimart.WithRetry(...)`
- `apimart.WithDebugLog(...)`
- `apimart.WithLogger(...)`

默认 Base URL：

```text
https://api.apimart.ai/v1/
```

## 日志与 Curl 调试

本 provider 默认会输出每次请求对应的 curl 命令，便于直接复现请求。

如果你还希望看到响应状态和响应 body，可以打开：

```go
svc := apimart.NewService("your-api-key",
    apimart.WithDebugLog(true),
)
```

如果要接入自己的日志系统：

```go
logger := log.New(os.Stdout, "[demo] ", log.LstdFlags)

svc := apimart.NewService("your-api-key",
    apimart.WithLogger(logger),
    apimart.WithDebugLog(true),
)
```

日志内容包括：

- curl command
- response status
- response body
- retry 日志（开启重试时）

## 支持的 Operation

```go
actions := svc.Actions("gpt-image-2")
// => []xai.Action{xai.GenImage, xai.EditImage}
```

```go
op, err := svc.Operation("gpt-image-2", xai.GenImage)
op, err := svc.Operation("gpt-image-2", xai.EditImage)
```

其它模型会返回 `xai.ErrNotFound`。

## 输入参数

通过 `Operation(...).Params().Set(...)` 设置：

| Param | 类型 | 必填 | 说明 |
|------|------|------|------|
| `Prompt` | `string` | 是 | 文生图 / 图生图提示词 |
| `Size` | `string` | 否 | 输出比例 |
| `Resolution` | `string` | 否 | 输出档位，`1k` / `2k` / `4k` |
| `OfficialFallback` | `bool` | 否 | 是否走官方兜底 |
| `Image` | `string` / `xai.Image` | 否 | 单张参考图 |
| `Images` | `[]string` / `[]xai.Image` / `[]any` | 图生图时必填 | 多张参考图 |

### Size 支持值

- `auto`
- `1:1`
- `3:2`
- `2:3`
- `4:3`
- `3:4`
- `5:4`
- `4:5`
- `16:9`
- `9:16`
- `2:1`
- `1:2`
- `21:9`
- `9:21`

### Resolution 支持值

- `1k`
- `2k`
- `4k`

### 参数校验规则

- `Prompt` 必填
- 图生图时必须提供 `Images` 或 `Image`
- 参考图最多 16 张
- `Resolution=4k` 仅支持这 6 个比例：
  - `16:9`
  - `9:16`
  - `2:1`
  - `1:2`
  - `21:9`
  - `9:21`

## 文生图示例

```go
ctx := context.Background()
svc := apimart.NewService(os.Getenv("APIMART_API_KEY"))

op, err := svc.Operation("gpt-image-2", xai.GenImage)
if err != nil {
    panic(err)
}

op.Params().
    Set("Prompt", "一只橘猫坐在窗台上看夕阳，水彩画风格").
    Set("Size", "16:9").
    Set("Resolution", "2k")

resp, err := xai.CallSync(ctx, svc, op, svc.Options())
if err != nil {
    panic(err)
}

fmt.Println("task_id:", resp.TaskID())

results, err := xai.Wait(ctx, svc, resp, func(resp xai.OperationResponse) {
    fmt.Println("polling:", resp.TaskID())
})
if err != nil {
    panic(err)
}

for i := 0; i < results.Len(); i++ {
    out := results.At(i).(*xai.OutputImage)
    fmt.Println(out.URL())
}
```

## 图生图示例

```go
op, err := svc.Operation("gpt-image-2", xai.EditImage)
if err != nil {
    panic(err)
}

op.Params().
    Set("Prompt", "把这张照片变成水彩插画风格，保留主体构图").
    Set("Size", "4:3").
    Set("Resolution", "2k").
    Set("OfficialFallback", true).
    Set("Images", []string{
        "https://example.com/photo.jpg",
    })
```

> 注意：APIMart 的图生图仍然走 `POST /v1/images/generations`，参考图通过 `image_urls` 传入。

## 异步任务模型

提交成功后会先拿到一个未完成的 `OperationResponse`：

```go
resp, err := xai.CallSync(ctx, svc, op, svc.Options())
taskID := resp.TaskID()
```

这个 `taskID` 可以落库，之后再恢复：

```go
resp2, err := xai.GetTask(ctx, svc, "gpt-image-2", xai.GenImage, taskID)
results, err := xai.Wait(ctx, svc, resp2, nil)
```

这正是本包相对同步图片 provider 的核心价值。

## 返回结果

任务完成后，`Results()` 返回 `xai.Results`，其中每一项都是 `*xai.OutputImage`：

```go
img := results.At(0).(*xai.OutputImage)
fmt.Println(img.URL())
fmt.Println(img.Image.Type())
```

结果 URL 来自任务查询响应中的：

```text
data.result.images[0].url[0]
```

## 错误处理

### 提交阶段 HTTP 错误

服务端返回非 2xx 时，会解析：

```json
{
  "error": {
    "code": 429,
    "message": "请求过于频繁，请稍后再试",
    "type": "rate_limit_error"
  }
}
```

并格式化成普通 Go error。

### 任务失败

轮询到失败态时，返回的 `OperationResponse` 会实现：

```go
xai.OperationResponseWithError
```

可通过：

```go
if errResp, ok := resp.(xai.OperationResponseWithError); ok {
    fmt.Println(errResp.GetError())
}
```

拿到任务失败原因。

`xai.Wait(...)` 在最终失败时也会把这个错误返回出来。

## 与 `spec/openai` 默认图片能力的区别

`spec/openai` 默认内建的 `openai/gpt-image-2` 图片操作是“同步结果模型”，调用后直接返回图片结果。

而 APIMart 这个 provider：

- 模型名不同：`gpt-image-2`
- 返回 `task_id`
- 需要轮询任务状态
- 图生图通过 `image_urls` 提交参考图
- 额外支持 `Resolution` / `OfficialFallback`

因此它没有直接复用默认的同步图片 operation，而是单独实现了一套 provider。

## 示例代码

可直接参考：

- [examples/apimart/main.go](../../../../examples/apimart/main.go)
- [examples/apimart/README.md](../../../../examples/apimart/README.md)

## 测试

本包包含提交、轮询、失败态、参数校验、URI 覆盖等测试：

```bash
cd spec/openai
go test ./provider/apimart
```
