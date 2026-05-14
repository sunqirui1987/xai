// Seedance 2.0 (Yunshi Cloud CATS) video generation example.
//
// Usage:
//
//	export YUNSHI_API_KEY=your-yunshi-api-key
//	go run ./examples/seedance_yunshi
//
// This example includes a real-person reference image by default. The provider will
// push the image to Yunshi/Volcengine material review, await Active status, and use
// asset://{asset_id} for video generation. Set YUNSHI_GROUP_ID only if you want to
// override Yunshi's default material group.
//
// The SDK-facing model remains the original Volc Ark model id. The Yunshi provider
// maps it to the endpoint id before submit:
//   - doubao-seedance-2-0-260128      -> ep-20260325195111-lsfzx
//   - doubao-seedance-2-0-fast-260128 -> ep-20260326172546-nzmk4
package main

import (
	"context"
	"fmt"
	"os"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/yunshi"
)

const realPortraitURL = "https://aitoken-public.qnaigc.com/example/generate-image/smile-woman.png"

func main() {
	apiKey := os.Getenv("YUNSHI_API_KEY")
	if apiKey == "" {
		fmt.Println("Set YUNSHI_API_KEY to call Yunshi Cloud Seedance (see spec/seedance/provider/yunshi/README.md).")
		os.Exit(1)
	}

	yunshi.Register(apiKey)
	ctx := context.Background()
	svc, err := xai.New(ctx, "seedance://")
	if err != nil {
		fmt.Println("xai.New:", err)
		os.Exit(1)
	}

	model := xai.Model(seedance.ModelDoubaoSeedance20Fast)
	op, err := svc.Operation(model, xai.GenVideo)
	if err != nil {
		fmt.Println("Operation:", err)
		os.Exit(1)
	}

	params := op.Params().(*seedance.Params)
	params.
		Set(seedance.ParamPrompt, "一位真实女性站在海边栈道上，微风吹动头发，镜头缓慢向前推进，夕阳金色光影洒在她脸上与海面上，人物保持自然表情和真实质感。").
		Set(seedance.ParamReferenceImageURLs, []string{realPortraitURL}).
		Set(seedance.ParamGenerateAudio, true).
		Set(seedance.ParamRatio, "16:9").
		Set(seedance.ParamDuration, 5).
		Set(seedance.ParamWatermark, false)

	if groupID := os.Getenv("YUNSHI_GROUP_ID"); groupID != "" {
		params.Set(yunshi.ParamAssetGroupID, groupID)
		fmt.Println("material auto-push enabled with YUNSHI_GROUP_ID")
	} else {
		fmt.Println("material auto-push enabled with Yunshi default group")
	}

	resp, err := xai.CallSync(ctx, svc, op, svc.Options())
	if err != nil {
		fmt.Println("CallSync:", err)
		os.Exit(1)
	}
	if !resp.Done() {
		fmt.Println("submitted, task_id:", resp.TaskID(), "(polling ~2s per round)")
	} else {
		fmt.Println("task_id:", resp.TaskID(), "(sync)")
	}

	results, err := xai.Wait(ctx, svc, resp, func(r xai.OperationResponse) {
		if !r.Done() {
			fmt.Println("polling...", r.TaskID())
		}
	})
	if err != nil {
		fmt.Println("Wait:", err)
		os.Exit(1)
	}
	for i := 0; i < results.Len(); i++ {
		v := results.At(i).(*xai.OutputVideo)
		fmt.Println("video:", v.Video.StgUri())
	}
}
