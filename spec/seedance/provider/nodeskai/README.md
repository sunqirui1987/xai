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

- `asset_group_id`
- `content`
- `callback_url`
- `return_last_frame`
- `execution_expires_after`
- `service_tier`
- `tools`
- `safety_identifier`
- `seed`

其中 `content` 支持直接传入 `[]any` 或 `[]map[string]any`，便于接入 NoDesk 的虚拟资产库或后续扩展字段，而不需要修改通用 `spec/seedance`。

## 图片素材前置上传

当调用 NoDesk Seedance 2.0 且请求里带有参考图时，本实现支持在提交视频任务前，先把图片走一遍 NoDesk 素材库：

1. 下载 `reference_images` / `reference_image_urls` 中的图片
2. 使用素材库 OAuth 凭证上传到指定素材组
3. 轮询素材状态直到不是 `Processing`
4. 用素材库返回的 URL 替换原始图片 URL
5. 再调用 `POST /v1/video/generate`

启用这条链路需要：

- `NODESKAI_API_KEY`，用于视频生成接口
- `NODESKAI_CLIENT_ID`
- `NODESKAI_CLIENT_SECRET`

默认情况下，本实现会自动查找名为 `默认素材组` 的素材组；不存在时会自动创建，然后继续上传参考图。

也就是说，只要请求参数里本来就带有参考图，provider 就会自动完成“建组/复用组 -> 上传 -> await -> 替换 URL”这条链路。

如果你想显式指定素材组，仍然可以通过 `asset_group_id` 参数覆盖默认行为。

## 鉴权说明

- 平台接口通过 OAuth2 `client_credentials` 获取 `access_token` 后，再以 `Authorization: Bearer <access_token>` 调用。
- 如果直接传字符串给 `NewService` / `Register`，这里传入的应当是 **access token**，不是 `client_secret`。
- `NODESKAI_API_KEY` 用于视频生成接口。
- `NODESKAI_CLIENT_ID` / `NODESKAI_CLIENT_SECRET` 用于图片素材上传、素材组查询/创建、素材状态轮询；素材平台会先用它们换取 token，再调用数字资产接口。
- 素材库接口如果要求项目用户隔离，还应配置 `NODESKAI_EXTERNAL_USER_ID`。
- 默认素材组名称可通过 `NODESKAI_DEFAULT_ASSET_GROUP_NAME` 调整；不配时使用 `默认素材组`。
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
    nodeskai.RegisterWithClientCredentials(
        os.Getenv("NODESKAI_API_KEY"),
        os.Getenv("NODESKAI_CLIENT_ID"),
        os.Getenv("NODESKAI_CLIENT_SECRET"),
    )
    ctx := context.Background()
    svc, _ := xai.New(ctx, "seedance://")
    op, _ := svc.Operation(xai.Model("doubao-seedance-2-0-fast-260128"), xai.GenVideo)
    op.Params().(*seedance.Params).
        Set(seedance.ParamPrompt, "海浪拍打礁石，日落时分金色光芒洒满海面").
        Set(seedance.ParamReferenceImageURLs, []string{"https://example.com/reference.png"}).
        Set(seedance.ParamRatio, "16:9").
        Set(seedance.ParamDuration, 5).
        Set("resolution", "720p")
    resp, _ := xai.CallSync(ctx, svc, op, svc.Options())
    _, _ = xai.Wait(ctx, svc, resp, nil)
}
```
