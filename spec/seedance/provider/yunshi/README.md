# Yunshi Cloud Seedance Provider

This package implements `seedance.Backend` for Yunshi Cloud CATS Seedance 2.0 APIs.

Endpoints:

- Create task: `POST /api/v3/contents/generations/tasks`
- Query task: `GET /api/v3/contents/generations/tasks/{task_id}`
- Push material: `POST /api/v3/contents/generations/materials/push-volcengine`
- Query material: `GET /api/v3/contents/generations/materials/{asset_id}/volcengine-status`

The SDK-facing model IDs stay aligned with Volc Ark:

- `seedance.ModelDoubaoSeedance20`: `doubao-seedance-2-0-260128`
- `seedance.ModelDoubaoSeedance20Fast`: `doubao-seedance-2-0-fast-260128`

The provider maps them before HTTP submit:

- `doubao-seedance-2-0-260128` -> `ep-20260325195111-lsfzx`
- `doubao-seedance-2-0-fast-260128` -> `ep-20260326172546-nzmk4`

## Usage

```go
package main

import (
	"context"
	"fmt"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/yunshi"
)

func main() {
	ctx := context.Background()
	svc := yunshi.NewService("") // reads YUNSHI_API_KEY

	op, _ := svc.Operation(xai.Model(seedance.ModelDoubaoSeedance20Fast), xai.GenVideo)
	op.Params().(*seedance.Params).
		Set(seedance.ParamPrompt, "老人走在大街上").
		Set(seedance.ParamReferenceImageURLs, []string{"https://example.com/actor_reference.jpg"}).
		Set(seedance.ParamGenerateAudio, true).
		Set(seedance.ParamRatio, "16:9").
		Set(seedance.ParamDuration, 4).
		Set(seedance.ParamWatermark, false)

	resp, err := op.Call(ctx)
	if err != nil {
		panic(err)
	}
	for i := 0; i < resp.Results().Len(); i++ {
		v := resp.Results().Index(i)
		fmt.Println(v.Video.StgUri())
	}
}
```

For real-person reference materials, the provider will push image/video/audio reference URLs, wait for `Active`, and submit `asset://{asset_id}` to the generation endpoint. Pass `yunshi.ParamAssetGroupID` or set `YUNSHI_GROUP_ID` only when you need to override Yunshi's default material group.

```go
op.Params().(*seedance.Params).
	Set(seedance.ParamReferenceImageURLs, []string{"https://example.com/actor_reference.jpg"}).
	Set(yunshi.ParamAssetGroupID, "your_group_id")
```

To pass Yunshi's raw `content` array directly, set `yunshi.ParamContent`.
