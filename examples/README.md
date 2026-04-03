# Examples

Runnable demos for multiple providers/models via the xai API.

## Quick Start

```bash
# Run Kling examples
go run ./examples/kling

# Run Audio examples (ASR/TTS)
go run ./examples/audio
go run ./examples/audio all

# List models, actions, and schema only
go run ./examples/kling models

# Run Veo examples
go run ./examples/veo
go run ./examples/veo all

# Run Sora examples
go run ./examples/sora
go run ./examples/sora all

# Run Vidu examples
go run ./examples/vidu/video
go run ./examples/vidu/video all

# Seedance (Volc Ark, needs ARK_API_KEY)
export ARK_API_KEY=your-key
go run ./examples/seedance

# Video: pick Volc Ark vs Qiniu Kling from model id (see examples/shared/video_by_model.go)
go run ./examples/videorouter doubao-seedance-2-0-260128
go run ./examples/videorouter kling-v2-5-turbo

# Run by model (Kling)
go run ./examples/kling kling-v2-1
go run ./examples/kling/images kling-v2-1
go run ./examples/kling/video kling-v2-6
```

## Backend Mode

- **Mock** (default): No API key needed. Returns placeholder URLs. Works in CI.
- **Real**: Set `QINIU_API_KEY` to use the Qnagic API for actual generation. Volc Ark Seedance uses `ARK_API_KEY` ([spec/seedance/provider/volc/README.md](../spec/seedance/provider/volc/README.md)).

```bash
export QINIU_API_KEY=your-key
go run ./examples/kling kling-v2-1
```

## Directory Structure

```
examples/
├── README.md
├── audio/
│   ├── README.md
│   ├── main.go
│   ├── service.go
│   ├── asr.go
│   ├── tts.go
│   └── list_voices.go
├── vidu/
│   ├── README.md
│   ├── output/
│   │   └── output.go
│   ├── shared/
│   │   └── service.go
│   └── video/
│       ├── main.go
│       ├── urls.go
│       ├── helpers.go
│       ├── call_sync_example.go
│       ├── vidu_q1_text_to_video.go
│       ├── vidu_q1_reference_urls.go
│       ├── vidu_q1_reference_subjects.go
│       ├── vidu_q1_reference_subjects_audio.go
│       ├── vidu_q2_text_to_video.go
│       ├── vidu_q2_reference_urls.go
│       ├── vidu_q2_reference_subjects.go
│       ├── vidu_q2_image_to_video_pro.go
│       ├── vidu_q2_image_to_video_pro_audio.go
│       ├── vidu_q2_image_to_video_turbo.go
│       └── vidu_q2_start_end_to_video_pro.go
├── sora/
│   ├── README.md
│   ├── main.go
│   └── urls.go
├── veo/
│   ├── README.md
│   ├── main.go
│   ├── veo_2_0_generate_001.go
│   ├── veo_2_0_generate_exp.go
│   ├── veo_2_0_generate_preview.go
│   ├── veo_3_0_generate_preview.go
│   ├── veo_3_0_fast_generate_preview.go
│   ├── veo_3_1_generate_preview.go
│   └── veo_3_1_fast_generate_preview.go
├── seedance/
│   └── main.go             # Volc Ark Seedance GenVideo
├── videorouter/
│   └── main.go             # GenVideo: provider from model (Seedance→Ark, Kling→Qiniu)
├── shared/
│   ├── service.go          # NewService (Kling + mock)
│   └── video_by_model.go   # VideoGenServiceForModel → Volc Seedance vs Qiniu Kling
└── kling/
    ├── main.go             # Dispatches to images/ and video/ by model
    ├── models.go           # RunModels: list models, actions, schema
    ├── example_test.go
    ├── images/
    │   ├── main.go
    │   ├── urls.go         # DemoImageURLs, printImageResults
    │   ├── call_sync_example.go   # CallSync + TaskID + GetTask
    │   ├── kling_v1.go
    │   ├── kling_v15.go
    │   ├── kling_v2.go
    │   ├── kling_v2_new.go
    │   ├── kling_v21.go
    │   └── kling_image_o1.go
    └── video/
        ├── main.go
        ├── urls.go         # DemoVideoURLs, printVideoResults
        ├── kling_v21.go
        ├── kling_v25_turbo.go
        ├── kling_v26.go
        ├── kling_video_o1.go
        ├── kling_v3.go
        └── kling_v3_omni.go
```

## Models

**Image models**: kling-v1, kling-v1-5, kling-v2, kling-v2-new, kling-v2-1, kling-image-o1

**Video models**: kling-v2-1, kling-v2-5-turbo, kling-v2-6, kling-video-o1, kling-v3, kling-v3-omni

**Veo models**: veo-2.0-generate-001, veo-2.0-generate-exp, veo-2.0-generate-preview, veo-3.0-generate-preview, veo-3.0-fast-generate-preview, veo-3.1-generate-preview, veo-3.1-fast-generate-preview

**Sora models**: sora-2, sora-2-pro

**Vidu models**: vidu-q1, vidu-q2, viduq2-pro, viduq2-turbo

**Seedance (Volc Ark)**: doubao-seedance-2-0-260128 (see [Ark 文档](https://www.volcengine.com/docs/82379/1520757?lang=zh))

**Audio models**: asr (ASR), tts-v1 (TTS)

## CallSync + TaskID

The `call-sync` demo shows async task persistence:

- `CallSync` starts the operation and returns resp
- `resp.TaskID()` gets the task ID to save to DB
- `xai.GetTask(ctx, svc, model, action, taskID)` restores OperationResponse from taskID
- `xai.Wait` polls until done

```bash
go run ./examples/kling/images call-sync
```

## Tests

```bash
go test ./examples/kling/... -v -run Example
```

## See Also

- [spec/kling/kling_image.md](../spec/kling/kling_image.md)
- [spec/kling/kling_video.md](../spec/kling/kling_video.md)
