# Volc Ark Provider（火山引擎 · 方舟）

本包实现 `seedance.Backend`，通过 **火山方舟** 推理域名调用「内容生成 / 视频生成」异步任务接口。

## 官方文档

| 主题 | 链接 |
|------|------|
| 创建视频生成任务（含 Seedance 2.0 请求体说明） | [创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh) |
| 查询单个任务状态与结果 | [查询视频生成任务 API](https://www.volcengine.com/docs/82379/1521309?lang=zh) |
| API Key、Bearer、`ark.cn-beijing.volces.com` 等 | [Base URL 及鉴权](https://www.volcengine.com/docs/82379/1298459?lang=zh) |

文档更新频繁，**联调与排错以官方页面为准**。下文「参数约定」概括官方语义；「本包实现」说明本仓库实际序列化的字段。

---

## 创建任务 API 速览（官方）

- **方法 / 路径**：`POST https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks`（区域以控制台为准，见 [Base URL 及鉴权](https://www.volcengine.com/docs/82379/1298459?lang=zh)）。
- **鉴权**：`Authorization: Bearer <API Key>`。
- **请求体**：JSON，至少包含 `model`（Model ID 或 Endpoint ID）与 `content`（多模态内容数组）。其余字段多为可选，按模型能力与支持范围生效；不符合模型时可能被忽略或报错（官方「强校验 / 弱校验」见下）。

---

## 参数约定（官方语义）

### `content`（必选数组）

表示输入给模型的信息，支持文本、图片、音频、视频、样片任务 ID 等块对象。常见**组合**包括（详见 [创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh)）：

- 仅文本；
- 文本（可选）+ 图片；
- 文本（可选）+ 视频（Seedance 2.0 / 2.0 fast）；
- 文本（可选）+ 图片 + 音频 / 视频 / 两者兼有（2.0 系列）；
- 基于「样片任务 ID」的 draft 流程（**仅 Seedance 1.5 pro**，与 `draft` 等字段配合）。

**注意（官方）**：Seedance 2.0 系列对含真人人脸的参考图/视频有直接上传限制；平台另有合规方案，以文档与教程为准。音频**不可单独**作为唯一输入，须至少带参考图或参考视频。

### 根级常用字段（节选）

| 字段 | 类型 | 默认值 / 说明（官方摘要） |
|------|------|---------------------------|
| `callback_url` | string | 可选。任务状态变更时方舟 POST 回调；回调体与查询任务接口返回结构一致。`status` 含 `queued` / `running` / `succeeded` / `failed` / `expired` 等。 |
| `return_last_frame` | boolean | 默认 `false`。为 `true` 时可通过查询接口取成片**尾帧 PNG**（与成片同分辨率、无水印），用于多段连续视频衔接。 |
| `service_tier` | string | 默认 `default`（在线）；`flex` 为离线推理、配额与计价不同。**Seedance 2.0 / 2.0 fast 不支持离线推理**。已提交任务不可改服务等级。 |
| `execution_expires_after` | int | 默认 `172800`（秒），从创建时间起算，范围 `[3600, 259200]`；超时任务标记为 `expired`。 |
| `generate_audio` | boolean | 默认 `true`。**Seedance 2.0 / 2.0 fast、1.5 pro** 等支持：是否生成与画面对齐的声音；有声成片为单声道。 |
| `draft` | boolean | 默认 `false`。**仅 1.5 pro**：样片模式；开启后多为 480p draft，且与尾帧、离线等等能力互斥，以文档为准。 |
| `tools` | object[] | **仅 2.0 / 2.0 fast**：工具配置。 |
| `safety_identifier` | string | 终端用户标识（建议哈希），≤64 字符，用于合规辅助。 |
| `resolution` | string | 分辨率枚举：`480p` / `720p` / `1080p`（部分模型/场景不支持某些组合，见官方「输出视频格式」表）。不同模型默认值不同（如 2.0 默认 `720p`，1.0 pro 系列默认 `1080p` 等）。 |
| `ratio` | string | 宽高比：`16:9`、`4:3`、`1:1`、`3:4`、`9:16`、`21:9`、`adaptive`（按输入自动选最近比例；**2.0 / 1.5 pro** 等支持规则见文档）。图生视频若与上传图比例不一致，平台可能**居中裁剪**。 |
| `duration` | int | 默认 **`5`**（秒）：**不显式传 `duration` 与 `frames` 时**由方舟使用该默认。与 `frames` **二选一即可**，**`frames` 优先**；整数秒场景建议用 `duration`。文档时长范围：**1.0** 系列 **`[2,12]`**；**1.5 pro** **`[4,12]`** 或 **`-1`**（智能整数秒）；**2.0 / 2.0 fast** **`[4,15]`** 或 **`-1`**。`-1` 时实际秒数以**查询任务**返回的 `duration` 为准，且与计费相关。 |
| `frames` | int | 与 `duration` 二选一；**2.0 / 2.0 fast、1.5 pro 官方文档标注暂不支持**。用于小数秒时长时通过帧数控制；帧率按 **24** 推算，取值需满足官方给出的 `25+4n` 等形式约束。 |
| `seed` | int | 默认 `-1`（随机）。相同请求下相同 seed **倾向**相似结果，不保证完全一致。 |
| `camera_fixed` | boolean | 默认 `false`。参考图场景不支持；**2.0 / 2.0 fast 暂不支持**。 |
| `watermark` | boolean | 默认 `false`：`false` 无水印，`true` 有水印。 |

### 新方式 vs 旧方式（`resolution` / `ratio` / `duration` / `frames` / `seed` / `camera_fixed` / `watermark`）

官方说明：这些参数既可：

- **新方式（推荐）**：放在**请求体根对象**与同级的 `model`、`content` 一起传入；**强校验**，错误易直接报错。
- **旧方式**：在**文本提示词末尾**追加 `--[parameters]`；**弱校验**，错误时可能静默回退默认值。

**本仓库 `seedance.Params` 只映射其中一部分根级字段**（`duration`、`ratio`、`generate_audio`、`watermark`）；其余根级能力请用官方 **`--` 后缀**写在 `text`/`prompt` 中，或在本包外自行构造 HTTP 请求体。

---

## 本包实现：`seedance.Params` → 请求体

[`backend.go`](backend.go) 中 `buildTaskBody` 当前行为：

| `seedance` 参数名 | 写入 Ark 的位置 | 说明 |
|-------------------|-----------------|------|
| `text` 或 `prompt` | `content[]` 中 `{ "type":"text", "text":"..." }` | 必选（`Operation.Call` 与 `buildTaskBody` 均要求有主文案）。 |
| `reference_image_urls` | `content[]` 多条 `{ "type":"image_url", "image_url":{"url":"..."}, "role":"reference_image" }` | 可为多条 URL 字符串或 JSON/代码中的 `[]string`。 |
| `reference_video_urls` | `content[]`，`type`=`video_url`，`role`=`reference_video` | 同上。 |
| `reference_audio_urls` | `content[]`，`type`=`audio_url`，`role`=`reference_audio` | 同上；须满足官方「不可仅音频」约束。 |
| `duration` | 根对象 `duration` | 经 [`seedance.ValidateVideoDuration`](../duration.go) 按 **model id** 校验区间与是否允许 `-1`；`0` 非法。不传则请求体不含该字段，方舟默认 **5s**。 |
| `ratio` | 根对象 `ratio` | 非空则写入，如 `16:9`、`adaptive`。 |
| `generate_audio` | 根对象 `generate_audio` | bool 或 Params 支持的字符串真假。 |
| `watermark` | 根对象 `watermark` | 同上。 |

参数常量见 [`spec/seedance/params.go`](../params.go)；包级说明见 [`spec/seedance/README.md`](../README.md)。

---

## HTTP 约定（实现）

- **Base URL**：默认 `https://ark.cn-beijing.volces.com`（可通过 `volc.WithBaseURL` 覆盖为文档列出的其它区域/端点）。
- **创建任务**：`POST /api/v3/contents/generations/tasks`，`Authorization: Bearer <API Key>`，与 [创建视频生成任务 API](https://www.volcengine.com/docs/82379/1520757?lang=zh) 一致。
- **查询任务**：优先 `GET /api/v3/contents/generations/tasks/{task_id}`；若返回 404，再尝试 `GET /api/v3/contents/generations/tasks?id={task_id}`（兼容部分网关/文档变体）。
- **创建响应中的任务 ID**：解析顶层 `id`，或 `data.id` / `data.task_id`（不同响应封装兼容）。
- **查询响应**：读取 `status`（大小写不敏感）；成功时从 `video_url`、`content`、嵌套 `data.output` 等位置提取成片地址（与常见 Ark JSON 结构对齐，详见 `backend.go`）。

---

## 环境变量

- `ARK_API_KEY`：`NewClient("")` / `NewService("")` 未传 key 时使用。
- `VOLC_MOCK_CURL`：非空时只打印 curl、不发起真实 HTTP（与七牛侧 `QINIU_MOCK_CURL` 类似，便于脚本探测）。

---

## 调试与日志（对齐 Kling Qiniu Client）

默认行为与 [`spec/kling/provider/qiniu/client.go`](../../../kling/provider/qiniu/client.go) 一致：

- 每次请求前向 logger 打印 **等价 curl**（含完整 `Authorization: Bearer …`，**勿复制到公开环境**）。
- `WithDebugLog(true)`（默认）时额外打印 HTTP **状态码**与 **响应体**（体长度超过约 4KB 会截断）。
- 关闭详细输出：`volc.NewService(key, volc.WithDebugLog(false))`（curl 仍会打印，除非将 `WithLogger` 设为 `log.New(io.Discard, "", 0)`）。
- 自定义输出：`WithLogger(myLogger)`。
- 可选重试：`WithRetry(n, baseDelay)`，对 **429** 与 **500/502/503/504** 以及网络错误做指数退避（`maxRetries` 为额外重试次数，默认 0）。

`seedance.Backend`（`backend.go`）在 **Submit / GetTaskStatus** 等步骤使用 `Client.LogDebug` 打任务 id、归一化 `status`、是否已解析 `video_url` 等，前缀为 `[volc]`。

---

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

---

## 依赖

- 规格包：[`spec/seedance`](../README.md)
- 成功时的 `xai.Results` 与 Vidu 共用 [`spec/vidu/video`](../../vidu/video) 中的 `OutputVideo` 构造逻辑，便于上层统一处理成片 URL。
