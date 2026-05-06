// Seedance (NoDesk AI) video generation example.
//
// NoDesk AI docs:
//   - spec/seedance/provider/nodeskai/NoDesk AI · 视频生成 API 接入指南(1).html
//
// Usage:
//
//	export NODESKAI_CLIENT_ID=ndapp_xxx
//	export NODESKAI_CLIENT_SECRET=your-secret
//	go run ./examples/seedance_nodeskai
package main

import (
	"context"
	"fmt"
	"os"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/nodeskai"
)

func main() {
	clientID := os.Getenv("NODESKAI_CLIENT_ID")
	clientSecret := os.Getenv("NODESKAI_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		fmt.Println("Set NODESKAI_CLIENT_ID and NODESKAI_CLIENT_SECRET to call NoDesk AI Seedance.")
		os.Exit(1)
	}

	nodeskai.RegisterWithClientCredentials(clientID, clientSecret)
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
		Set(seedance.ParamPrompt, "夕阳下的海边栈道，镜头缓慢向前推进，金色光影映在海面上。").
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
