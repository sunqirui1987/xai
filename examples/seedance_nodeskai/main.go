// Seedance (NoDesk AI) video generation example.
//
// NoDesk AI docs:
//   - spec/seedance/provider/nodeskai/NoDesk AI · 视频生成 API 接入指南(1).html
//
// Usage:
//
//	export NODESKAI_API_KEY=your-video-api-key
//	export NODESKAI_CLIENT_ID=ndapp_xxx
//	export NODESKAI_CLIENT_SECRET=your-secret
//	go run ./examples/seedance_nodeskai
//
// This example includes a real-person reference image by default. The provider will
// automatically create or reuse the default asset group, upload the image, await it,
// and use the returned asset URL for video generation.
package main

import (
	"context"
	"fmt"
	"os"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/nodeskai"
)

const realPortraitURL = "https://aitoken-public.qnaigc.com/example/generate-image/smile-woman.png"

func main() {
	apiKey := os.Getenv("NODESKAI_API_KEY")
	clientID := os.Getenv("NODESKAI_CLIENT_ID")
	clientSecret := os.Getenv("NODESKAI_CLIENT_SECRET")
	if apiKey == "" {
		fmt.Println("Set NODESKAI_API_KEY for NoDesk AI video generation.")
		os.Exit(1)
	}
	if clientID == "" || clientSecret == "" {
		fmt.Println("Set NODESKAI_CLIENT_ID and NODESKAI_CLIENT_SECRET for NoDesk AI image asset auth.")
		os.Exit(1)
	}

	nodeskai.RegisterWithClientCredentials(apiKey, clientID, clientSecret)
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
		Set("resolution", "720p")

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
