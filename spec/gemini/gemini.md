# spec/gemini — Gemini Unified Interface Specification

`spec/gemini` is the unified interface layer for Google Gemini models in the xai project. It defines capabilities based on `google.golang.org/genai` and supports multiple backend implementations (official API, OpenAI-compatible gateways, Qiniu Cloud, etc.).

## 1. Overview

| Item | Description |
| --- | --- |
| Package path | `github.com/goplus/xai/spec/gemini` |
| Dependency | `google.golang.org/genai` |
| Capabilities | `Gen` / `GenStream` (chat), `Operation` (image/video) |
| Default schemes | `gemini` (official), `gemini-qiniu` (Qiniu Provider) |

## 2. Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  xai.Service (Gen / GenStream / Operation)                   │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│  gemini.Service                                              │
│  - Features: Gen | GenStream | Operation                     │
│  - Backend: implements GenerateContent / GenerateImages / EditImage │
└─────────────────────────────┬───────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐   ┌───────────────────┐   ┌───────────────────┐
│ genai official│   │ provider/qiniu     │   │ Other providers    │
│ (Gemini API)  │   │ (OpenAI-compatible)│   │                    │
└───────────────┘   └───────────────────┘   └───────────────────┘
```

### 2.1 Backend Interface

`Backend` defines the capabilities required by backend implementations:

| Method | Purpose |
| --- | --- |
| `Actions(model)` | Returns the list of actions supported by the model |
| `GenerateContent` | Non-streaming chat |
| `GenerateContentStream` | Streaming chat |
| `GenerateImages` | Text-to-image generation |
| `EditImage` | Image editing / fusion |
| `GenerateVideosFromSource` | Video generation (optional) |
| `GetVideosOperation` | Get video task status (optional) |
| `RecontextImage` / `UpscaleImage` / `SegmentImage` | Optional |

### 2.2 Dual-path Design (Qiniu Provider)

`spec/gemini/provider/qiniu` uses a dual-path design:

- **Gen / GenStream**: Uses OpenAI-compatible `POST /v1/chat/completions` for text-to-image, image-to-image, and streaming output
- **Operation**: `GenImage` uses `POST /v1/images/generations`, `EditImage` uses `POST /v1/images/edits`

## 3. Quick Start

### 3.1 Using Official Gemini API

```go
import (
    "context"
    xai "github.com/goplus/xai/spec"
)

func main() {
    // Set GEMINI_API_KEY env var or pass via URI
    svc, err := xai.New(context.Background(), "gemini:base=https://generativelanguage.googleapis.com/&key=your_api_key")
    if err != nil {
        panic(err)
    }
    // Use svc.Gen / svc.GenStream / svc.Operation ...
}
```

### 3.2 Using Qiniu Provider

```go
import (
    "context"
    xai "github.com/goplus/xai/spec"
    "github.com/goplus/xai/spec/gemini/provider/qiniu"
)

func main() {
    // After Register, xai.New can resolve the service by scheme
    qiniu.Register("your_token") // or leave empty to use QINIU_API_KEY
    svc, err := xai.New(context.Background(), "gemini-qiniu:")
    if err != nil {
        panic(err)
    }
    _ = svc
}
```

### 3.3 Creating Service Directly (without Register)

```go
import (
    "github.com/goplus/xai/spec/gemini"
    "github.com/goplus/xai/spec/gemini/provider/qiniu"
)

func main() {
    svc := qiniu.NewService("your_token")
    // Or with options
    svc = qiniu.NewService("", qiniu.WithBaseURL(qiniu.OverseasBaseURL))
}
```

## 4. Capability Examples

### 4.1 Text-to-Image (Streaming / Non-streaming)

```go
params := svc.Params().
    Model(xai.Model("gemini-2.5-flash-image")).
    Messages(svc.UserMsg().Text("Draw a cute orange cat sitting on a windowsill watching the sunset"))

resp, err := svc.Gen(ctx, params, nil)
// Or streaming
for r, err := range svc.GenStream(ctx, params, nil) {
    // Handle each chunk
}
```

### 4.2 Image-to-Image (Multimodal)

```go
params := svc.Params().
    Model(xai.Model("gemini-3.0-pro-image-preview")).
    Messages(svc.UserMsg().
        Text("Convert this image to watercolor style").
        ImageFromStgUri(xai.ImageJPEG, "https://example.com/photo.jpg"))

resp, err := svc.Gen(ctx, params, nil)
```

### 4.3 Operation: Text-to-Image

```go
op, err := svc.Operation(xai.Model("gemini-2.5-flash-image"), xai.GenImage)
if err != nil {
    return err
}
op.Params().
    Set("Prompt", "A fairy cottage in a dreamy forest, surrounded by magical light").
    Set("AspectRatio", "16:9")

resp, err := op.Call(ctx, svc, nil)
// resp.Results() to get generated images
```

### 4.4 Operation: Image Editing

```go
op, err := svc.Operation(xai.Model("gemini-2.5-flash-image"), xai.EditImage)
if err != nil {
    return err
}
img := svc.ImageFromStgUri(xai.ImageJPEG, "https://example.com/input.jpg")
ref, _ := svc.ReferenceImage(img, 0, xai.RawReferenceImage)

op.Params().
    Set("Prompt", "Convert this image to oil painting style").
    Set("References", []genai.ReferenceImage{ref.(genai.ReferenceImage)}).
    Set("AspectRatio", "16:9")

resp, err := op.Call(ctx, svc, nil)
```

## 5. Qiniu Provider Configuration

### 5.1 ClientOptions

| Option | Description |
| --- | --- |
| `WithBaseURL(url)` | Set API endpoint, default `https://api.qnaigc.com/v1/` |
| `WithHTTPClient(cli)` | Custom HTTP client |
| `WithRetry(maxRetries, baseDelay)` | Retry with exponential backoff; 429/5xx auto-retry |
| `WithDebugLog(enabled)` | Debug logging (curl command, response status, etc.) |
| `WithLogger(logger)` | Custom logger |

### 5.2 Default Endpoints

| Constant | Description |
| --- | --- |
| `DefaultBaseURL` | Domestic `https://api.qnaigc.com/v1/` |
| `OverseasBaseURL` | Overseas `https://openai.sufy.com/v1/` |

### 5.3 URI Overrides

```go
qiniu.Register("default_token")
// Use different key
svc, _ := xai.New(ctx, "gemini-qiniu:key=xxx")
// Use overseas endpoint
svc, _ := xai.New(ctx, "gemini-qiniu:base=https://openai.sufy.com/v1/&key=xxx")
```

## 6. Model List (Qiniu)

| Model | Description |
| --- | --- |
| `gemini-2.5-flash-image` | Text-to-image, image-to-image, image editing |
| `gemini-3.0-pro-image-preview` | Same as above |
| `gemini-3.1-flash-image-preview` | Same as above |

## 7. Example Project

Run `examples/gemini`:

```bash
# Set environment variable
export QINIU_API_KEY=your_token

# List available demos
go run ./examples/gemini

# Run specific demo
go run ./examples/gemini chat-text
go run ./examples/gemini chat-image
go run ./examples/gemini chat-tool
go run ./examples/gemini image-generate
go run ./examples/gemini image-edit

# Streaming mode
go run ./examples/gemini --stream chat-text
```

## 8. Detailed API Reference

For the full interface design and curl examples of Qiniu Gemini Provider, see:

- [provider/qiniu/gemini.md](provider/qiniu/gemini.md)

Includes request/response formats and examples for Chat Completions, Images Generations, and Images Edits.

## 9. References

- [Gemini Chat Completions (397191373e0)](https://apidocs.qnaigc.com/397191373e0)
- [Gemini Images Generations (397191374e0)](https://apidocs.qnaigc.com/397191374e0)
- [Gemini Images Edits (397191375e0)](https://apidocs.qnaigc.com/397191375e0)
- [google.golang.org/genai](https://pkg.go.dev/google.golang.org/genai)
