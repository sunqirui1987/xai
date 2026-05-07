# xai

[中文文档](./readme-zh.md)

Unified Go SDK for chat, image, video, and audio across leading AI models and providers.

Build once, switch models when you need to, and handle long-running generation jobs without rebuilding your integration layer every time.

## Why teams use xai

- One Go SDK for chat, multimodal input, image, video, and audio
- One integration surface across multiple model families and providers
- Built for product workflows, not just one-off demos
- First-class support for async generation with `TaskID`, polling, and resume
- Runnable examples that shorten time from evaluation to production

## Supported model families

| Family | Typical capabilities |
| --- | --- |
| OpenAI-compatible | Chat, multimodal chat, tool calling, image and video input |
| Gemini | Chat, tool calling, image generation, image editing, video |
| Kling | Image generation, video generation |
| Seedance | Video generation |
| Sora | Video generation |
| Veo | Video generation |
| Vidu | Video generation |
| Audio | ASR, TTS |

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
export QINIU_API_KEY=your-key
export ARK_API_KEY=your-ark-key
export NODESKAI_ACCESS_TOKEN=your-token
export NODESKAI_CLIENT_ID=your-client-id
export NODESKAI_CLIENT_SECRET=your-client-secret
```

### Run examples

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

## Why this is useful in real products

Most AI SDKs are optimized for one modality or one provider. Product teams usually need more:

- one model for chat
- another for image generation
- another for video generation
- a reliable way to manage async jobs

`xai` gives you a more stable integration layer so model choices can change without forcing your application architecture to change with them.

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
