# xai

[English](./README.md)

面向 Go 的统一 AI SDK，覆盖聊天、图片、视频、音频，以及多模型、多 provider 的接入场景。

写一套集成代码，就能在不同模型之间切换；遇到视频这类长任务，也不用再为轮询、恢复、状态管理单独补一层。

## 为什么团队会选 xai

- 一套 Go SDK，同时覆盖 chat、多模态、image、video、audio
- 一套接入方式，适配多个模型族和 provider
- 更适合产品化流程，而不只是临时 demo
- 原生支持异步生成任务：`TaskID`、轮询、恢复
- 自带可运行 examples，缩短从评估到上线的路径

## 已支持的模型族

| 模型族 | 典型能力 |
| --- | --- |
| OpenAI-compatible | Chat、多模态 chat、tool calling、图片和视频输入 |
| Gemini | Chat、tool calling、图片生成、图片编辑、视频 |
| Kling | 图片生成、视频生成 |
| Seedance | 视频生成 |
| Sora | 视频生成 |
| Veo | 视频生成 |
| Vidu | 视频生成 |
| Audio | ASR、TTS |

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
export QINIU_API_KEY=your-key
export ARK_API_KEY=your-ark-key
export NODESKAI_ACCESS_TOKEN=your-token
export NODESKAI_CLIENT_ID=your-client-id
export NODESKAI_CLIENT_SECRET=your-client-secret
```

### 直接跑示例

```bash
# OpenAI-compatible chat
go run ./examples/openai text
go run ./examples/openai image
go run ./examples/openai function-call

# Gemini
go run ./examples/gemini chat-text
go run ./examples/gemini image-generate

# Kling
go run ./examples/kling/images kling-v2-1
go run ./examples/kling/video kling-v2-6

# Sora
go run ./examples/sora text-to-video

# Veo
go run ./examples/veo veo-3.0-generate-preview

# Vidu
go run ./examples/vidu/video q2-image-pro-audio

# Seedance
go run ./examples/seedance
go run ./examples/seedance_qiniu
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

## 它为什么更适合真实产品

很多 AI SDK 只优化了一个方向，或者只适合一个 provider。但真正的产品通常需要：

- 一个模型做 chat
- 一个模型做图片
- 一个模型做视频
- 一套稳定的异步任务处理机制

`xai` 的价值就是把这层集成抽象稳定下来，让你可以换模型，而不用频繁重写业务接入层。

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
