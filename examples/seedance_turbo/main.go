// Seedance (Turbo/Yoofang Dreamina) video generation example.
//
// Turbo/Yoofang docs:
//   - POST /v1/video/generations — create task
//   - GET /v1/video/generations/{task_id} — query task
//
// Usage:
//
//	export TURBO_API_KEY=your-turbo-api-key
//	go run ./examples/seedance_turbo
//
// This example includes a public AIGC reference video by default. The Turbo
// provider uploads it to the Yoofang asset library and sends it as
// `reference_video_urls`.
package main

import (
	"context"
	"fmt"
	"os"

	xai "github.com/goplus/xai/spec"
	"github.com/goplus/xai/spec/seedance"
	"github.com/goplus/xai/spec/seedance/provider/turbo"
)

const referenceVideoURL = "https://aitoken-public.qnaigc.com/example/generate-video/the-little-dog-is-running-on-the-lawn.mp4"

func main() {
	if os.Getenv("TURBO_API_KEY") == "" {
		fmt.Println("Set TURBO_API_KEY to call Turbo/Yoofang Seedance (see spec/seedance/provider/turbo/README.md).")
		os.Exit(1)
	}

	turbo.Register(os.Getenv("TURBO_API_KEY"))
	ctx := context.Background()
	svc, err := xai.New(ctx, "seedance://")
	if err != nil {
		fmt.Println("xai.New:", err)
		os.Exit(1)
	}

	model := xai.Model(seedance.ModelDreaminaSeedance20)
	op, err := svc.Operation(model, xai.GenVideo)
	if err != nil {
		fmt.Println("Operation:", err)
		os.Exit(1)
	}

	op.Params().(*seedance.Params).
		Set(seedance.ParamPrompt, "7-second cinematic fantasy animation, use the main subject from the reference video as the hero, keep the same identity, proportions, and recognizable features. The hero stands in a magical forest, facing a giant cute fluffy monster. The hero looks brave, holds a glowing wooden sword and a round shield with golden runes. Warm golden magic light bursts from the sword and gently pushes the monster backward. The monster is not scary, not hurt, big expressive eyes, soft fur, tiny horns. The monster transforms into colorful sparkling stars and soft magical dust. The hero smiles proudly at the end. Magical forest, glowing mushrooms, fireflies, sunset light beams, soft mist, cinematic camera movement, Pixar-like 3D cartoon animation, storybook fantasy style, wholesome heroic mood, vibrant colors, volumetric lighting").
		Set(seedance.ParamReferenceVideoURLs, []string{referenceVideoURL}).
		Set(seedance.ParamRatio, "9:16").
		Set(seedance.ParamDuration, 7).
		Set(seedance.ParamGenerateAudio, true).
		Set("resolution", "1080p")

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
