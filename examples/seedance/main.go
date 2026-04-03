// Seedance (Volc Ark) video generation example.
//
// Official docs:
//   - https://www.volcengine.com/docs/82379/1520757?lang=zh — create task
//   - https://www.volcengine.com/docs/82379/1521309?lang=zh — query task
//
// Usage:
//
//	export ARK_API_KEY=your-ark-api-key
//	go run ./examples/seedance
//
// To switch provider by model (Seedance vs Kling video), see examples/videorouter.
package main

import (
	"context"
	"fmt"
	"os"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/volc"
)

func main() {
	if os.Getenv("ARK_API_KEY") == "" {
		fmt.Println("Set ARK_API_KEY to call Volc Ark (see spec/seedance/provider/volc/README.md).")
		os.Exit(1)
	}

	volc.Register(os.Getenv("ARK_API_KEY"))
	ctx := context.Background()
	svc, err := xai.New(ctx, "seedance://")
	if err != nil {
		fmt.Println("xai.New:", err)
		os.Exit(1)
	}

	model := xai.Model(seedance.ModelDoubaoSeedance20)
	op, err := svc.Operation(model, xai.GenVideo)
	if err != nil {
		fmt.Println("Operation:", err)
		os.Exit(1)
	}

	op.Params().(*seedance.Params).
		Set(seedance.ParamPrompt, "A short product clip, cinematic lighting.").
		Set(seedance.ParamRatio, "16:9").
		Set(seedance.ParamDuration, 5).
		Set(seedance.ParamGenerateAudio, false).
		Set(seedance.ParamWatermark, false)

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
			fmt.Println("polling…", r.TaskID())
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
