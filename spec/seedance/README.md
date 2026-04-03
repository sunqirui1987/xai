# Seedance（火山方舟视频生成）

本包定义 **Seedance / 豆包视频生成** 在 xai 中的规格层：URI scheme、`GenVideo` 操作、参数名与异步结果类型。HTTP 接入由 [`provider/volc`](provider/volc/README.md) 实现。

官方文档（设计与联调时请以文档为准）：

- [创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh)（Seedance 2.0 等）
- [查询视频生成任务 API](https://www.volcengine.com/docs/82379/1521309?lang=zh)
- [Base URL 及鉴权](https://www.volcengine.com/docs/82379/1298459?lang=zh)

## 职责划分

| 组件 | 说明 |
|------|------|
| `seedance.Scheme` (`seedance://`) | 通过 `seedance.Register(svc)` 注册到 `xai.New` |
| `Service` + `Backend` | 仅支持 `FeatureOperation`；`GenVideo` 提交任务并轮询 |
| `Params` | 映射到 Ark 请求体字段；也可用 `ark_content_json` 传入与文档一致的完整 `content` 数组 |

## 与 Ark 请求体的对应关系

Ark 创建任务体核心字段为 `model`、`content`（多模态块数组），以及可选的 `generate_audio`、`ratio`、`duration`、`watermark` 等（见[创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh)）。

| xai 参数名 | Ark JSON 字段 / 行为 |
|------------|----------------------|
| `text` 或 `prompt` | 在 `content` 中追加 `{ "type":"text", "text":"..." }`（与 `ark_content_json` 互斥） |
| `reference_image_urls` | 多条 `{ "type":"image_url", "image_url":{"url":"..."}, "role":"reference_image" }` |
| `reference_video_urls` | `type`=`video_url`，`role`=`reference_video` |
| `reference_audio_urls` | `type`=`audio_url`，`role`=`reference_audio` |
| `duration` / `ratio` / `generate_audio` / `watermark` | 同名字段写入请求根对象 |
| `ark_content_json` | **整段替换** `content` 数组（JSON 字符串，需为数组）；用于与官方示例完全一致的高级用法 |

## 任务状态

查询接口返回的 `status` 在文档中多为排队 / 运行中 / 成功 / 失败等英文枚举；实现侧会做大小写无关归一化，并在 `succeeded`（及兼容的 `completed` 等）时解析 `video_url`（含嵌套在 `content`、`data.output` 等常见形态）。细节以[查询视频生成任务 API](https://www.volcengine.com/docs/82379/1521309?lang=zh)为准。

异步任务创建后 `OperationResponse.Done()` 为 false，需用 `xai.Wait` 轮询；`AsyncOperationResponse` 默认每次 `Retry` 前 `Sleep` 约 **2 秒**。调用 `xai.Wait(ctx, svc, resp, progress)` 时建议传入非 nil 的 `progress`，在长时间生成时打印 `TaskID()` 等，避免终端长时间无输出（见 `examples/videorouter`、`examples/seedance`）。

## 代码入口

```go
import (
    "github.com/goplus/xai/spec/seedance"
    "github.com/goplus/xai/spec/seedance/provider/volc"
)

volc.Register(os.Getenv("ARK_API_KEY"))
// xai.New(ctx, "seedance://")
```

完整示例见仓库 [`examples/seedance`](../../examples/seedance)。  
按 **模型 id 自动选 Volc Ark 或七牛 Kling** 的 GenVideo 示例：[`examples/videorouter`](../../examples/videorouter)（实现见 [`examples/shared/video_by_model.go`](../../examples/shared/video_by_model.go)）。
