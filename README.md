# xai

[中文文档](./readme-zh.md)

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)](#quick-start)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)
[![Examples](https://img.shields.io/badge/Examples-Runnable-success)](./examples/README.md)
[![Models](https://img.shields.io/badge/Models-Chat%20%7C%20Image%20%7C%20Video%20%7C%20Audio-orange)](#capability-matrix)

One Go SDK for chat, image, video, and audio across leading AI models and providers.

It provides a common Go interface for multimodal generation, tool calling, and long-running async jobs.

## Why xai

Projects that use multiple models usually end up maintaining separate request parameters, polling code, and result handling for each provider.

`xai` tries to keep that surface smaller:

- One SDK for chat, multimodal input, image, video, and audio
- One integration surface across multiple model families and providers
- First-class support for async generation with `TaskID`, polling, and resume
- Runnable examples

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

## Capability matrix

| Family | Chat | Image | Video | Audio | Async tasks |
| --- | --- | --- | --- | --- | --- |
| OpenAI-compatible | Yes | Input / related workflows | Input / related workflows | No | Partial |
| Gemini | Yes | Yes | Yes | No | Yes |
| Kling | No | Yes | Yes | No | Yes |
| Seedance | No | No | Yes | No | Yes |
| Sora | No | No | Yes | No | Yes |
| Veo | No | No | Yes | No | Yes |
| Vidu | No | No | Yes | No | Yes |
| Audio | No | No | No | Yes | Yes |

## Supported models and multimodal capabilities

### OpenAI-compatible

- Chat models: `gemini-3.0-pro-preview`, `deepseek/deepseek-v3.2-251201`, and other OpenAI-compatible chat models exposed by the provider
- Multimodal input:
  - text
  - image URL input
  - video URL input
  - file-style video ID input such as `qfile-...`
  - multiple video inputs in one conversation
- Tool calling:
  - `tool_use` -> local execution -> `tool_result`
- Image operations:
  - `openai/gpt-image-2` for generation and edit

### Gemini

- Chat and multimodal chat:
  - text
  - image input
  - tool calling
- Image models:
  - `gemini-2.5-flash-image`
  - `gemini-3.0-pro-image-preview`
  - `gemini-3.1-flash-image-preview`
- Image capabilities:
  - text-to-image
  - image edit
- Video models through the Gemini provider:
  - `veo-2.0-generate-001`
  - `veo-2.0-generate-exp`
  - `veo-2.0-generate-preview`
  - `veo-3.0-generate-preview`
  - `veo-3.0-fast-generate-preview`
  - `veo-3.1-generate-preview`
  - `veo-3.1-fast-generate-preview`

### Veo multimodal video

- Text-to-video
- Image-to-video
- First-frame + last-frame video
- Video-to-video style input flow
- Multi-reference images
  - supported in `veo-2.0-generate-exp` and `veo-3.1-generate-preview`

### Kling

- Image models:
  - `kling-v1`
  - `kling-v1-5`
  - `kling-v2`
  - `kling-v2-new`
  - `kling-v2-1`
  - `kling-image-o1`
- Image capabilities:
  - text-to-image
  - image-to-image
  - reference image workflows
- Video models:
  - `kling-v2-1`
  - `kling-v2-5-turbo`
  - `kling-v2-6`
  - `kling-v2-7`
  - `kling-v2-8`
  - `kling-v2-9`
  - `kling-video-o1`
  - `kling-v3`
  - `kling-v3-omni`
- Video capabilities:
  - text-to-video
  - image-to-video
  - keyframe video with first and end frame
  - multi-reference image and video inputs on selected models

### Sora

- Models:
  - `sora-2`
  - `sora-2-pro`
- Video capabilities:
  - text-to-video
  - image-to-video
  - remix from an existing source video

### Seedance

- Models:
  - `doubao-seedance-2-0-260128`
  - other `doubao-seedance-*` IDs recognized by the provider flow
- Multimodal video inputs:
  - text prompt
  - reference images
  - reference videos
  - reference audio
- Video capabilities:
  - text-to-video
  - prompt-guided video generation with multimodal references

### Vidu

- Models:
  - `vidu-q1`
  - `vidu-q2`
  - `viduq2-turbo`
  - `viduq2-pro`
  - `viduq3-turbo`
  - `viduq3-pro`
- Video capabilities:
  - text-to-video
  - reference-to-video
  - image-to-video
  - start-end-to-video
  - optional audio generation on supported flows

### Audio

- Models:
  - `asr`
  - `tts-v1`
- Audio capabilities:
  - speech-to-text
  - text-to-speech
  - voice listing with provider support

## What you can build

- Chat applications with text, image, and video input
- Tool-calling workflows with request and result round-trips
- Text-to-image and image editing pipelines
- Text-to-video and image-to-video pipelines
- First-frame, last-frame, multi-reference, and remix video workflows
- Speech-to-text and text-to-speech services
- Long-running generation flows that survive process restarts

## Quick start

### Install

```bash
go get github.com/goplus/xai
```

### Set credentials

```bash
# Qiniu API key
# Get it from: https://portal.qiniu.com/ai-inference/api-key
export QINIU_API_KEY=your-key
```

### Try it in 30 seconds

```bash
# OpenAI-compatible chat
go run ./examples/openai text

# Image generation
go run ./examples/gemini image-generate

# Video generation
go run ./examples/sora text-to-video
```

### Explore more examples

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
```

More runnable demos: [examples/README.md](./examples/README.md)

## Minimal examples

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

### Video generation

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

## Notes

- Most video and image generation flows use the operation API and may return async tasks
- Examples in this repository use Qiniu by default
- The README focuses on the common entry points; provider-specific details are documented under `spec/*` and `examples/*`

## Good fit for

- Go teams shipping AI features into existing products
- Platform teams standardizing access to multiple model providers
- Teams building image, video, and audio workflows in one backend
- Teams that want a cleaner path from prototype to production

## Example coverage

- `examples/openai`: text, multimodal input, function calling, thinking
- `examples/gemini`: chat, tool use, image generation, image editing
- `examples/kling/images`: text-to-image and image-to-image
- `examples/kling/video`: text-to-video and image-to-video
- `examples/sora`: text-to-video, image-to-video, remix
- `examples/veo`: multi-version video generation, first and last frame, multi-reference, video input
- `examples/vidu/video`: Q1, Q2, Pro, and Turbo video workflows
- `examples/audio`: ASR and TTS
- `examples/seedance`: Seedance integrations
- `examples/videorouter`: route video generation by model ID

## License

Apache-2.0
