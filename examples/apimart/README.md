# APIMart GPT-Image-2 Examples

Runnable demos for `spec/openai/provider/apimart`.

## Prerequisites

- Go `1.24.5+`
- `APIMART_API_KEY`
- Network access to `https://api.apimart.ai`

## Quick Start

```bash
export APIMART_API_KEY=your-key

# List demos
go run ./examples/apimart

# Text-to-image
go run ./examples/apimart generate

# Image-to-image
go run ./examples/apimart edit
```

## Resume an Existing Task

```bash
export APIMART_API_KEY=your-key
export APIMART_TASK_ID=task_01KPQ7J7DWB7QZ3WCEK3YVPBRA

go run ./examples/apimart resume
```

## What It Demonstrates

- `apimart.NewService(...)`
- `svc.Operation("gpt-image-2", xai.GenImage|xai.EditImage)`
- async submit via `xai.CallSync`
- persist `resp.TaskID()`
- resume polling via `xai.GetTask`
- wait for final image URLs via `xai.Wait`

