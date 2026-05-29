// Seedance (Qiniu) assets review example.
//
// Usage:
//
//	export QINIU_API_KEY=your-qiniu-api-key
//	go run ./examples/seedance_qiniu_assets create-group "我的虚拟人像分组" "用于存放公司形象虚拟人像素材"
//	go run ./examples/seedance_qiniu_assets create https://example.com/portrait.jpg "年轻男人" [qgroupid]
//	go run ./examples/seedance_qiniu_assets get qasset-uid001-1716100100000000000
//	go run ./examples/seedance_qiniu_assets await qasset-uid001-1716100100000000000
//
// Optional:
//
//	export QINIU_ASSETS_BASE_URL=https://openai.sufy.com
//	export QINIU_ASSETS_MODEL=bytedance/doubao-seedance-2-0-260128
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
	assetsqiniu "github.com/goplus/xai/spec/seedance_assets/provider/qiniu"
)

const (
	defaultModel         = "bytedance/doubao-seedance-2-0-260128"
	defaultAwaitTimeout  = 5 * time.Minute
	defaultAwaitInterval = 2 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	apiKey := strings.TrimSpace(os.Getenv("QINIU_API_KEY"))
	if apiKey == "" {
		fmt.Println("Set QINIU_API_KEY to call Qiniu assets API.")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	opts := []assetsqiniu.ClientOption{
		assetsqiniu.WithDebugLog(true),
	}
	if baseURL := strings.TrimSpace(os.Getenv("QINIU_ASSETS_BASE_URL")); baseURL != "" {
		opts = append(opts, assetsqiniu.WithBaseURL(baseURL))
	}
	svc := assetsqiniu.NewService(apiKey, opts...)
	ctx := context.Background()

	switch os.Args[1] {
	case "create-group":
		name := "我的虚拟人像分组"
		description := "用于存放公司形象虚拟人像素材"
		if len(os.Args) >= 3 {
			name = os.Args[2]
		}
		if len(os.Args) >= 4 {
			description = os.Args[3]
		}
		if err := runCreateGroup(ctx, svc, name, description); err != nil {
			fmt.Println("create-group:", err)
			os.Exit(1)
		}
	case "create":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		assetURL := os.Args[2]
		name := "年轻男人"
		if len(os.Args) >= 4 {
			name = os.Args[3]
		}
		groupID := ""
		if len(os.Args) >= 5 {
			groupID = os.Args[4]
		}
		if err := runCreateAsset(ctx, svc, assetURL, name, groupID); err != nil {
			fmt.Println("create:", err)
			os.Exit(1)
		}
	case "get":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		if err := runGet(ctx, svc, os.Args[2]); err != nil {
			fmt.Println("get:", err)
			os.Exit(1)
		}
	case "await":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		if err := runAwait(ctx, svc, os.Args[2]); err != nil {
			fmt.Println("await:", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func runCreateGroup(ctx context.Context, svc *assetsqiniu.Service, name, description string) error {
	group, err := svc.CreateGroup(ctx, &seedanceassets.CreateAssetGroupRequest{
		Name:        name,
		Description: description,
		Type:        "aigc",
		Model:       model(),
	})
	if err != nil {
		return err
	}
	fmt.Println("qgroupid:", group.ID)
	fmt.Println("type:", group.Type)
	fmt.Println("name:", group.Name)
	fmt.Println("description:", group.Description)
	fmt.Println("model:", group.Model)
	fmt.Println("status:", group.Status)
	fmt.Println("is_default:", group.IsDefault)
	fmt.Println("fail_reason:", group.FailReason)
	fmt.Println("created_at:", group.CreatedAt)
	fmt.Println("updated_at:", group.UpdatedAt)
	return nil
}

func runCreateAsset(ctx context.Context, svc *assetsqiniu.Service, assetURL, name, groupID string) error {
	ret, err := svc.UploadAsset(ctx, &seedanceassets.UploadAssetRequest{
		GroupID:   groupID,
		Name:      name,
		AssetType: "image",
		URL:       assetURL,
		Model:     model(),
	})
	if err != nil {
		return err
	}
	fmt.Println("qassetid:", ret.AssetID)
	fmt.Println("type:", ret.AssetType)
	fmt.Println("name:", ret.Name)
	fmt.Println("model:", ret.Model)
	fmt.Println("status:", ret.Status)
	fmt.Println("group_id:", ret.GroupID)
	fmt.Println("fail_reason:", ret.FailReason)
	fmt.Println("created_at:", ret.CreatedAt)
	fmt.Println("updated_at:", ret.UpdatedAt)
	fmt.Println("asset_ref:", "qasset://"+ret.AssetID)
	return nil
}

func runGet(ctx context.Context, svc *assetsqiniu.Service, assetID string) error {
	asset, err := svc.GetAsset(ctx, assetID)
	if err != nil {
		return err
	}
	printAsset(asset)
	return nil
}

func runAwait(ctx context.Context, svc *assetsqiniu.Service, assetID string) error {
	awaitCtx, cancel := context.WithTimeout(ctx, defaultAwaitTimeout)
	defer cancel()

	attempt := 0
	for {
		attempt++
		asset, err := svc.GetAsset(awaitCtx, assetID)
		if err != nil {
			return err
		}
		log.Printf("[await] attempt=%d qassetid=%q status=%q", attempt, asset.ID, asset.Status)
		if !isReviewing(asset.Status) {
			printAsset(asset)
			if asset.ID != "" {
				fmt.Println("asset_ref:", "qasset://"+asset.ID)
			}
			return nil
		}

		select {
		case <-awaitCtx.Done():
			return awaitCtx.Err()
		case <-time.After(defaultAwaitInterval):
		}
	}
}

func printAsset(asset *seedanceassets.Asset) {
	fmt.Println("qassetid:", asset.ID)
	fmt.Println("type:", asset.Type)
	fmt.Println("name:", asset.Name)
	fmt.Println("model:", asset.Model)
	fmt.Println("status:", asset.Status)
	fmt.Println("group_id:", asset.GroupID)
	fmt.Println("fail_reason:", asset.FailReason)
	fmt.Println("created_at:", asset.CreatedAt)
	fmt.Println("updated_at:", asset.UpdatedAt)
}

func isReviewing(status string) bool {
	st := strings.TrimSpace(strings.ToLower(status))
	return st == seedanceassets.AssetStatusPending || st == seedanceassets.AssetStatusReviewing
}

func model() string {
	if m := strings.TrimSpace(os.Getenv("QINIU_ASSETS_MODEL")); m != "" {
		return m
	}
	return defaultModel
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./examples/seedance_qiniu_assets create-group [name] [description]")
	fmt.Println("  go run ./examples/seedance_qiniu_assets create image_url [name] [qgroupid]")
	fmt.Println("  go run ./examples/seedance_qiniu_assets get qassetid")
	fmt.Println("  go run ./examples/seedance_qiniu_assets await qassetid")
}
