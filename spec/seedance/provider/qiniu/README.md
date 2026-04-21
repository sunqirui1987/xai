# Qiniu Provider（七牛 Qnagic）

本包实现 `seedance.Backend`，通过七牛 / Qnagic 的 `contents/generations/tasks` 接口调用 Seedance 视频生成。

## API

- 创建任务：`POST /v3/contents/generations/tasks`
- 查询任务：`GET /v3/contents/generations/tasks/{task_id}`
- 鉴权：`Authorization: Bearer <QINIU_API_KEY>`
- 默认 Base URL：`https://api.qnaigc.com`

参考文档：

- [创建任务](https://apidocs.qnaigc.com/439818050e0)
- [查询任务](https://apidocs.qnaigc.com/439818051e0)

## 本包映射

`seedance.Params` 到 Qiniu 请求体的映射如下：

- `text` / `prompt` -> `content[]` 的首个 `{type:"text"}`
- `ratio` -> 根级 `ratio`
- `duration` -> 根级 `duration`
- `generate_audio` -> 根级 `generate_audio`
- `reference_image_urls` -> 若有第 1 张，则映射为 `first_frame`；若有第 2 张，则映射为 `last_frame`

兼容行为：

- `seedance` 规格层本身没有 `resolution`、`first_frame_image_url`、`last_frame_image_url` 这些字段，所以 qiniu 实现没有扩展 spec，只用现有参数做最小映射。
- 因此当前 qiniu provider 不支持从通用 `seedance.Params` 显式传 `resolution`；如果后续要支持，建议放在 provider 私有层而不是改通用 spec。

当前 Qiniu Seedance 后端不会发送 `watermark`、`reference_video_urls`、`reference_audio_urls`，因为你给的 Qnagic Seedance 文档示例没有展示这些字段。

模型名约定：

- Qiniu 实际请求使用 `bytedance/doubao-seedance-2-0-260128`
- 为了不改通用 `spec/seedance`，provider 内部会把 `doubao-seedance-2-0-260128` 自动补成带 `bytedance/` 前缀的模型名

## 使用

```go
import (
    "context"
    "os"

    xai "github.com/goplus/xai/spec"
    "github.com/goplus/xai/spec/seedance"
    "github.com/goplus/xai/spec/seedance/provider/qiniu"
)

func main() {
    qiniu.Register(os.Getenv("QINIU_API_KEY"))

    ctx := context.Background()
    svc, _ := xai.New(ctx, "seedance://")
    op, _ := svc.Operation(xai.Model(seedance.ModelDoubaoSeedance20), xai.GenVideo)
    op.Params().(*seedance.Params).
        Set(seedance.ParamPrompt, "夕阳下的城市街道，电影感镜头缓慢推进").
        Set(seedance.ParamRatio, "16:9").
        Set(seedance.ParamDuration, 5).
        Set(seedance.ParamGenerateAudio, true)
    _, _ = xai.CallSync(ctx, svc, op, svc.Options())
}
```
