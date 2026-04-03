# Volc Ark Provider（火山引擎 · 方舟）

本包实现 `seedance.Backend`，通过 **火山方舟** 推理域名调用「内容生成 / 视频生成」异步任务接口。

## 官方文档

| 主题 | 链接 |
|------|------|
| 创建视频生成任务（含 Seedance 2.0 请求体说明） | [创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh) |
| 查询单个任务状态与结果 | [查询视频生成任务 API](https://www.volcengine.com/docs/82379/1521309?lang=zh) |
| API Key、Bearer、`ark.cn-beijing.volces.com` 等 | [Base URL 及鉴权](https://www.volcengine.com/docs/82379/1298459?lang=zh) |

文档更新频繁，**联调与排错以官方页面为准**；本 README 描述的是本仓库实现与文档的对应关系。

## HTTP 约定（实现）

- **Base URL**：默认 `https://ark.cn-beijing.volces.com`（可通过 `volc.WithBaseURL` 覆盖为文档列出的其它区域/端点）。
- **创建任务**：`POST /api/v3/contents/generations/tasks`，`Authorization: Bearer <API Key>`，与[创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh)一致。
- **查询任务**：优先 `GET /api/v3/contents/generations/tasks/{task_id}`；若返回 404，再尝试 `GET /api/v3/contents/generations/tasks?id={task_id}`（兼容部分网关/文档变体）。
- **创建响应中的任务 ID**：解析顶层 `id`，或 `data.id` / `data.task_id`（不同响应封装兼容）。
- **查询响应**：读取 `status`（大小写不敏感）；成功时从 `video_url`、`content`、嵌套 `data.output` 等位置提取成片地址（与常见 Ark JSON 结构对齐，详见 `backend.go`）。

## 环境变量

- `ARK_API_KEY`：`NewClient("")` / `NewService("")` 未传 key 时使用。
- `VOLC_MOCK_CURL`：非空时只打印 curl、不发起真实 HTTP（与七牛侧 `QINIU_MOCK_CURL` 类似，便于脚本探测）。

## 调试与日志（对齐 Kling Qiniu Client）

默认行为与 [`spec/kling/provider/qiniu/client.go`](../../../kling/provider/qiniu/client.go) 一致：

- 每次请求前向 logger 打印 **等价 curl**（含完整 `Authorization: Bearer …`，**勿复制到公开环境**）。
- `WithDebugLog(true)`（默认）时额外打印 HTTP **状态码**与 **响应体**（体长度超过约 4KB 会截断）。
- 关闭详细输出：`volc.NewService(key, volc.WithDebugLog(false))`（curl 仍会打印，除非将 `WithLogger` 设为 `log.New(io.Discard, "", 0)`）。
- 自定义输出：`WithLogger(myLogger)`。
- 可选重试：`WithRetry(n, baseDelay)`，对 **429** 与 **500/502/503/504** 以及网络错误做指数退避（`maxRetries` 为额外重试次数，默认 0）。

`seedance.Backend`（`backend.go`）在 **Submit / GetTaskStatus** 等步骤使用 `Client.LogDebug` 打任务 id、归一化 `status`、是否已解析 `video_url` 等，前缀为 `[volc]`。

## 使用方式

```go
import (
    "context"
    "os"

    xai "github.com/goplus/xai/spec"
    "github.com/goplus/xai/spec/seedance"
    "github.com/goplus/xai/spec/seedance/provider/volc"
)

func main() {
    volc.Register(os.Getenv("ARK_API_KEY"))
    ctx := context.Background()
    svc, _ := xai.New(ctx, "seedance://")
    op, _ := svc.Operation(xai.Model(seedance.ModelDoubaoSeedance20), xai.GenVideo)
    op.Params().(*seedance.Params).
        Set(seedance.ParamPrompt, "…").
        Set(seedance.ParamRatio, "16:9").
        Set(seedance.ParamDuration, 11).
        Set(seedance.ParamGenerateAudio, true).
        Set(seedance.ParamWatermark, false)
    resp, _ := xai.CallSync(ctx, svc, op, svc.Options())
    results, _ := xai.Wait(ctx, svc, resp, nil)
    _ = results
}
```

与官方 cURL 完全一致的 `content` 数组可设置 `seedance.ParamArkContentJSON` 为 **JSON 数组字符串**，见 [`spec/seedance/README.md`](../README.md)。

## 依赖

- 规格包：[`spec/seedance`](../README.md)
- 成功时的 `xai.Results` 与 Vidu 共用 [`spec/vidu/video`](../../vidu/video) 中的 `OutputVideo` 构造逻辑，便于上层统一处理成片 URL。
