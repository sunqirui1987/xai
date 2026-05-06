# NoDesk AI Provider

本包实现 `seedance.Backend`，通过 **NoDesk AI** 的 Seedance 2.0 视频生成接口提供后端服务。

## API 约定

- Base URL：`https://llm-gateway-api.nodesk.tech`
- 提交任务：`POST /v1/video/generate`
- 查询任务：`GET /v1/video/generate/{task_id}`
- 鉴权：`Authorization: Bearer <access_token>`

数字资产库能力已拆分到独立包：[`spec/seedance_assets/provider/nodeskai`](../../../seedance_assets/provider/nodeskai)。

## 已实现参数映射

标准 `seedance.Params` 字段：

- `text` / `prompt` → `content[]` 中的 `{"type":"text","text":"..."}`
- `reference_images` / `reference_image_urls` → `image_url`
- `reference_video_urls` → `video_url`
- `reference_audio_urls` → `audio_url`
- `ratio` / `duration` / `resolution` / `generate_audio` / `watermark` → 根级字段

Provider 私有透传字段：

- `content`
- `callback_url`
- `return_last_frame`
- `execution_expires_after`
- `service_tier`
- `tools`
- `safety_identifier`
- `seed`

其中 `content` 支持直接传入 `[]any` 或 `[]map[string]any`，便于接入 NoDesk 的虚拟资产库或后续扩展字段，而不需要修改通用 `spec/seedance`。

## 鉴权说明

- 平台接口通过 OAuth2 `client_credentials` 获取 `access_token` 后，再以 `Authorization: Bearer <access_token>` 调用。
- 如果直接传字符串给 `NewService` / `Register`，这里传入的应当是 **access token**，不是 `client_secret`。
- `NODESKAI_MOCK_CURL`：非空时只打印 curl，不发真实请求

## 使用示例

```go
import (
    "context"
    "os"

    xai "github.com/goplus/xai/spec"
    "github.com/goplus/xai/spec/seedance"
    "github.com/goplus/xai/spec/seedance/provider/nodeskai"
)

func main() {
    nodeskai.Register(os.Getenv("NODESKAI_ACCESS_TOKEN"))
    ctx := context.Background()
    svc, _ := xai.New(ctx, "seedance://")
    op, _ := svc.Operation(xai.Model("doubao-seedance-2-0-fast-260128"), xai.GenVideo)
    op.Params().(*seedance.Params).
        Set(seedance.ParamPrompt, "海浪拍打礁石，日落时分金色光芒洒满海面").
        Set(seedance.ParamRatio, "16:9").
        Set(seedance.ParamDuration, 5).
        Set("resolution", "720p")
    resp, _ := xai.CallSync(ctx, svc, op, svc.Options())
    _, _ = xai.Wait(ctx, svc, resp, nil)
}
```
