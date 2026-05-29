// Seedance (Qiniu/Qnagic) video generation example.
//
// Qnagic docs:
//   - https://apidocs.qnaigc.com/439818050e0 — create task
//   - https://apidocs.qnaigc.com/439818051e0 — query task
//
// Usage:
//
//	export QINIU_API_KEY=your-qiniu-api-key
//	go run ./examples/seedance_qiniu
//
// This example includes a real-person reference image by default, matching the
// Yunshi example. The Qiniu provider sends it as `reference_image_urls`.
package main

import (
	"context"
	"fmt"
	"os"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/qiniu"
)

const realPortraitURL = "https://aitoken-public.qnaigc.com/example/generate-image/smile-woman.png"

func main() {
	if os.Getenv("QINIU_API_KEY") == "" {
		fmt.Println("Set QINIU_API_KEY to call Qiniu Seedance (see spec/seedance/provider/qiniu/README.md).")
		os.Exit(1)
	}

	qiniu.Register(os.Getenv("QINIU_API_KEY"))
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
		Set(seedance.ParamPrompt, "一位真实女性站在海边栈道上，微风吹动头发，镜头缓慢向前推进，夕阳金色光影洒在她脸上与海面上，人物保持自然表情和真实质感。").
		Set(seedance.ParamReferenceImageURLs, []string{realPortraitURL}).
		Set(seedance.ParamRatio, "16:9").
		Set(seedance.ParamDuration, 5).
		Set(seedance.ParamGenerateAudio, true)

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
