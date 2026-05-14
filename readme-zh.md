# xai

[English](./README.md)

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)](#快速开始)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)
[![Examples](https://img.shields.io/badge/Examples-可直接运行-success)](./examples/README.md)
[![Models](https://img.shields.io/badge/Models-Chat%20%7C%20Image%20%7C%20Video%20%7C%20Audio-orange)](#能力矩阵)

面向 Go 的统一 AI SDK，覆盖聊天、图片、视频、音频，以及多模型、多 provider 的接入场景。

它提供了一套统一的 Go 接口，用来处理多模态生成、工具调用和长时间异步任务。

## 为什么是 xai

一个项目如果同时接多个模型，通常就要分别维护请求参数、轮询逻辑和结果处理代码。

`xai` 主要做的是把这部分接口收敛起来：

- 一套 SDK，同时覆盖 chat、多模态、image、video、audio
- 一套接入方式，适配多个模型族和 provider
- 原生支持异步生成任务：`TaskID`、轮询、恢复
- 自带可运行 examples

```go
ctx := context.Background()
svc := qiniu.NewService(os.Getenv("QINIU_API_KEY"))

// chat
resp, _ := svc.Gen(ctx, svc.Params().
	Model("gemini-3.0-pro-preview").
	Messages(svc.UserMsg().Text("hello")), nil)

// video
op, _ := svc.Operation("sora-2", xai.GenVideo)
op.Params().Set("Prompt", "a cat running").Set("Seconds", "4")
results, _ := xai.Call(ctx, svc, op, svc.Options(), nil)
```

## 能力矩阵

| 模型族 | Chat | Image | Video | Audio | Async tasks |
| --- | --- | --- | --- | --- | --- |
| OpenAI-compatible | Yes | 输入与相关流程 | 输入与相关流程 | No | Partial |
| Gemini | Yes | Yes | Yes | No | Yes |
| Kling | No | Yes | Yes | No | Yes |
| Seedance | No | No | Yes | No | Yes |
| Sora | No | No | Yes | No | Yes |
| Veo | No | No | Yes | No | Yes |
| Vidu | No | No | Yes | No | Yes |
| Audio | No | No | No | Yes | Yes |

## 支持的模型与多模态能力

### OpenAI-compatible

- Chat 模型：
  - `gemini-3.0-pro-preview`
  - `deepseek/deepseek-v3.2-251201`
  - 以及 provider 暴露的其他 OpenAI-compatible chat 模型
- 多模态输入：
  - 文本
  - 图片 URL 输入
  - 视频 URL 输入
  - `qfile-...` 这类文件 ID 视频输入
  - 单轮会话中的多视频输入
- Tool calling：
  - `tool_use` -> 本地执行 -> `tool_result`
- 图片操作：
  - `openai/gpt-image-2`，支持生成与编辑

### Gemini

- Chat 与多模态 chat：
  - 文本
  - 图片输入
  - tool calling
- 图片模型：
  - `gemini-2.5-flash-image`
  - `gemini-3.0-pro-image-preview`
  - `gemini-3.1-flash-image-preview`
- 图片能力：
  - 文生图
  - 图片编辑
- 通过 Gemini provider 接入的视频模型：
  - `veo-2.0-generate-001`
  - `veo-2.0-generate-exp`
  - `veo-2.0-generate-preview`
  - `veo-3.0-generate-preview`
  - `veo-3.0-fast-generate-preview`
  - `veo-3.1-generate-preview`
  - `veo-3.1-fast-generate-preview`

### Veo 多模态视频能力

- 文生视频
- 图生视频
- 首帧 + 尾帧视频生成
- 引用视频输入的视频生成流程
- 多参考图视频生成
  - 当前支持 `veo-2.0-generate-exp` 和 `veo-3.1-generate-preview`

### Kling

- 图片模型：
  - `kling-v1`
  - `kling-v1-5`
  - `kling-v2`
  - `kling-v2-new`
  - `kling-v2-1`
  - `kling-image-o1`
- 图片能力：
  - 文生图
  - 图生图
  - 参考图工作流
- 视频模型：
  - `kling-v2-1`
  - `kling-v2-5-turbo`
  - `kling-v2-6`
  - `kling-v2-7`
  - `kling-v2-8`
  - `kling-v2-9`
  - `kling-video-o1`
  - `kling-v3`
  - `kling-v3-omni`
- 视频能力：
  - 文生视频
  - 图生视频
  - 首尾帧关键帧视频
  - 部分模型支持多参考图 / 视频输入

### Sora

- 模型：
  - `sora-2`
  - `sora-2-pro`
- 视频能力：
  - 文生视频
  - 图生视频
  - 基于已有视频的 remix

### Seedance

- 模型：
  - `doubao-seedance-2-0-260128`
  - 以及 provider 流程中识别的其他 `doubao-seedance-*` 模型
- 多模态视频输入：
  - 文本 prompt
  - 参考图片
  - 参考视频
  - 参考音频
- 视频能力：
  - 文生视频
  - 带多模态参考输入的视频生成

### Vidu

- 模型：
  - `vidu-q1`
  - `vidu-q2`
  - `viduq2-turbo`
  - `viduq2-pro`
  - `viduq3-turbo`
  - `viduq3-pro`
- 视频能力：
  - 文生视频
  - reference-to-video
  - 图生视频
  - 首尾帧视频生成
  - 部分流程支持音频生成

### Audio

- 模型：
  - `asr`
  - `tts-v1`
- 音频能力：
  - 语音转文本
  - 文本转语音
  - 在 provider 支持下可查询 voice 列表

## 你可以拿它做什么

- 构建支持文本、图片、视频输入的聊天应用
- 搭建工具调用与结果回传工作流
- 搭建文生图和图片编辑流程
- 搭建文生视频和图生视频流程
- 支持首帧、尾帧、多参考图、remix 等视频能力
- 提供语音转文本和文本转语音服务
- 把长时间生成任务稳定接入业务流程

## 快速开始

### 安装

```bash
go get github.com/goplus/xai
```

### 配置凭证

```bash
# 七牛 API Key
# 获取地址：https://portal.qiniu.com/ai-inference/api-key
export QINIU_API_KEY=your-key
```

### 30 秒先跑起来

```bash
# OpenAI-compatible chat
go run ./examples/openai text

# 图片生成
go run ./examples/gemini image-generate

# 视频生成
go run ./examples/sora text-to-video
```

### 继续看更多示例

```bash
# OpenAI-compatible chat
go run ./examples/openai image
go run ./examples/openai function-call

# Gemini
go run ./examples/gemini chat-text

# Kling
go run ./examples/kling/images kling-v2-1
go run ./examples/kling/video kling-v2-6

# Veo
go run ./examples/veo veo-3.0-generate-preview

# Vidu
go run ./examples/vidu/video q2-image-pro-audio

# Seedance
go run ./examples/seedance
go run ./examples/seedance_qiniu
go run ./examples/seedance_yunshi
```

更多可运行示例见 [examples/README.md](./examples/README.md)。

## 最小示例

### Chat

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/goplus/xai/spec/openai/provider/qiniu"
)

func main() {
	ctx := context.Background()
	svc := qiniu.NewService(os.Getenv("QINIU_API_KEY"))

	resp, err := svc.Gen(ctx, svc.Params().
		Model("gemini-3.0-pro-preview").
		Messages(svc.UserMsg().Text("What is the Sun?")), nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Len())
}
```

### 视频生成

```go
package main

import (
	"context"
	"fmt"
	"os"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/openai/provider/qiniu"
)

func main() {
	ctx := context.Background()
	svc := qiniu.NewService(os.Getenv("QINIU_API_KEY"))

	op, err := svc.Operation("sora-2", xai.GenVideo)
	if err != nil {
		panic(err)
	}

	op.Params().
		Set("Prompt", "A cat walking on the beach at sunset").
		Set("Seconds", "4")

	resp, err := xai.CallSync(ctx, svc, op, svc.Options())
	if err != nil {
		panic(err)
	}

	results, err := xai.Wait(ctx, svc, resp, nil)
	if err != nil {
		panic(err)
	}

	fmt.Println(results.Len())
}
```

## 说明

- 大多数图片和视频生成流程走 operation API，并且可能返回异步任务
- 仓库中的 examples 默认以七牛接入为主
- README 只保留通用入口，provider 细节放在 `spec/*` 和 `examples/*` 下

## 适合这些团队

- 在现有 Go 产品里落地 AI 功能的团队
- 需要统一多个模型 provider 接入方式的平台团队
- 要把图片、视频、音频能力做进同一个后端的团队
- 希望从原型平滑过渡到生产环境的团队

## 示例覆盖范围

- `examples/openai`: 文本、多模态、function calling、thinking
- `examples/gemini`: chat、tool use、图片生成、图片编辑
- `examples/kling/images`: 文生图、图生图
- `examples/kling/video`: 文生视频、图生视频
- `examples/sora`: text-to-video、image-to-video、remix
- `examples/veo`: 多版本视频生成、首尾帧、多参考图、视频输入
- `examples/vidu/video`: Q1、Q2、Pro、Turbo 视频流程
- `examples/audio`: ASR、TTS
- `examples/seedance`: Seedance 接入示例
- `examples/videorouter`: 按 model ID 路由视频生成

## License

Apache-2.0
