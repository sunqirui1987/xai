// Seedance (NoDesk AI) digital assets example.
//
// Usage:
//
//	export NODESKAI_CLIENT_ID=ndapp_xxx
//	export NODESKAI_CLIENT_SECRET=your-secret
//	go run ./examples/seedance_nodeskai_assets create-group "默认素材组" "Seedance 2.0 默认素材组"
//	go run ./examples/seedance_nodeskai_assets upload /path/to/portrait.jpg grp_abc123 "女性正脸-01"
//	  # upload command will await the uploaded asset_id automatically
//	go run ./examples/seedance_nodeskai_assets get asset_789xyz
//	go run ./examples/seedance_nodeskai_assets await asset_789xyz
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	seedanceassets "github.com/goplus/xai/spec/seedance_assets"
	assetsnodeskai "github.com/goplus/xai/spec/seedance_assets/provider/nodeskai"
)

const (
	defaultAwaitTimeout  = 5 * time.Minute
	defaultAwaitInterval = 2 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	start := time.Now()
	clientID := os.Getenv("NODESKAI_CLIENT_ID")
	clientSecret := os.Getenv("NODESKAI_CLIENT_SECRET")
	externalUserID := os.Getenv("NODESKAI_EXTERNAL_USER_ID")
	log.Printf("[boot] seedance_nodeskai_assets started cwd=%q args=%q", mustGetwd(), os.Args)
	log.Printf("[boot] env client_id_set=%t client_secret_set=%t", clientID != "", clientSecret != "")
	log.Printf("[boot] env NODESKAI_CLIENT_ID=%q len=%d", clientID, len(clientID))
	log.Printf("[boot] env NODESKAI_CLIENT_SECRET=%q len=%d", clientSecret, len(clientSecret))
	log.Printf("[boot] env NODESKAI_EXTERNAL_USER_ID=%q len=%d", externalUserID, len(externalUserID))
	if clientID == "" || clientSecret == "" {
		fmt.Println("Set NODESKAI_CLIENT_ID and NODESKAI_CLIENT_SECRET to call NoDesk AI digital assets API.")
		os.Exit(1)
	}
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	log.Printf("[boot] constructing assets service client_id=%s", maskClientID(clientID))
	svc := assetsnodeskai.NewServiceWithClientCredentials(clientID, clientSecret)
	ctx := context.Background()

	switch os.Args[1] {
	case "create-group":
		name := "默认素材组"
		description := "Seedance 2.0 默认素材组"
		if len(os.Args) >= 3 {
			name = os.Args[2]
		}
		if len(os.Args) >= 4 {
			description = os.Args[3]
		}
		log.Printf("[cmd] create-group requested name=%q description=%q", name, description)
		if err := runCreateGroup(ctx, svc, name, description); err != nil {
			log.Printf("[error] create-group failed after %s: %v", time.Since(start).Round(time.Millisecond), err)
			fmt.Println("create-group:", err)
			os.Exit(1)
		}
	case "upload":
		if len(os.Args) < 4 {
			printUsage()
			os.Exit(1)
		}
		filePath := os.Args[2]
		groupID := os.Args[3]
		name := ""
		if len(os.Args) >= 5 {
			name = os.Args[4]
		}
		log.Printf("[cmd] upload requested file=%q group_id=%q name=%q", filePath, groupID, name)
		if err := runUpload(ctx, svc, filePath, groupID, name); err != nil {
			log.Printf("[error] upload failed after %s: %v", time.Since(start).Round(time.Millisecond), err)
			fmt.Println("upload:", err)
			os.Exit(1)
		}
	case "get":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		log.Printf("[cmd] get requested asset_id=%q", os.Args[2])
		if err := runGet(ctx, svc, os.Args[2]); err != nil {
			log.Printf("[error] get failed after %s: %v", time.Since(start).Round(time.Millisecond), err)
			fmt.Println("get:", err)
			os.Exit(1)
		}
	case "await":
		if len(os.Args) < 3 {
			printUsage()
			os.Exit(1)
		}
		log.Printf("[cmd] await requested asset_id=%q", os.Args[2])
		if err := runAwait(ctx, svc, os.Args[2]); err != nil {
			log.Printf("[error] await failed after %s: %v", time.Since(start).Round(time.Millisecond), err)
			fmt.Println("await:", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}

	log.Printf("[done] command=%q elapsed=%s", os.Args[1], time.Since(start).Round(time.Millisecond))
}

func runCreateGroup(ctx context.Context, svc *assetsnodeskai.Service, name, description string) error {
	stepStart := time.Now()
	log.Printf("[group] calling CreateGroup name=%q description=%q", name, description)
	group, err := svc.CreateGroup(ctx, &seedanceassets.CreateAssetGroupRequest{
		Name:        name,
		Description: description,
	})
	if err != nil {
		return err
	}
	log.Printf("[group] CreateGroup succeeded elapsed=%s group_id=%q status=%q", time.Since(stepStart).Round(time.Millisecond), group.ID, group.Status)
	fmt.Println("group_id:", group.ID)
	fmt.Println("name:", group.Name)
	fmt.Println("description:", group.Description)
	fmt.Println("status:", group.Status)
	fmt.Println("asset_count:", group.AssetCount)
	fmt.Println("create_time:", group.CreateTime)
	return nil
}

func runUpload(ctx context.Context, svc *assetsnodeskai.Service, filePath, groupID, name string) error {
	stepStart := time.Now()
	log.Printf("[upload] open file path=%q", filePath)
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	log.Printf("[upload] file opened size=%d mode=%s mod_time=%s", info.Size(), info.Mode(), info.ModTime().Format(time.RFC3339))

	displayName := name
	if strings.TrimSpace(displayName) == "" {
		displayName = filepath.Base(filePath)
	}
	log.Printf("[upload] request summary group_id=%q file_name=%q display_name=%q", groupID, filepath.Base(filePath), displayName)

	req := &seedanceassets.UploadAssetRequest{
		GroupID:  groupID,
		Name:     name,
		FileName: filepath.Base(filePath),
		File:     f,
	}
	log.Printf("[upload] calling UploadAsset...")
	ret, err := svc.UploadAsset(ctx, req)
	if err != nil {
		return err
	}
	log.Printf("[upload] UploadAsset succeeded elapsed=%s asset_id=%q status=%q type=%q", time.Since(stepStart).Round(time.Millisecond), ret.AssetID, ret.Status, ret.AssetType)
	fmt.Println("success:", ret.Success)
	fmt.Println("asset_id:", ret.AssetID)
	fmt.Println("asset_type:", ret.AssetType)
	fmt.Println("status:", ret.Status)
	fmt.Println("tos_url:", ret.TOSURL)
	fmt.Println("file_name:", ret.FileName)
	fmt.Println("file_size:", ret.FileSize)

	if strings.TrimSpace(ret.AssetID) == "" {
		log.Printf("[upload] upload response missing asset_id, skip follow-up get")
		return nil
	}

	fmt.Println()
	log.Printf("[upload] follow up with AwaitAsset asset_id=%q", ret.AssetID)
	return runAwait(ctx, svc, ret.AssetID)
}

func runGet(ctx context.Context, svc *assetsnodeskai.Service, assetID string) error {
	stepStart := time.Now()
	log.Printf("[get] calling GetAsset asset_id=%q", assetID)
	asset, err := svc.GetAsset(ctx, assetID)
	if err != nil {
		return err
	}
	log.Printf("[get] GetAsset succeeded elapsed=%s asset_id=%q status=%q type=%q group_id=%q", time.Since(stepStart).Round(time.Millisecond), asset.ID, asset.Status, asset.Type, asset.GroupID)
	fmt.Println("id:", asset.ID)
	fmt.Println("group_id:", asset.GroupID)
	fmt.Println("name:", asset.Name)
	fmt.Println("type:", asset.Type)
	fmt.Println("status:", asset.Status)
	fmt.Println("url:", asset.URL)
	fmt.Println("create_time:", asset.CreateTime)
	return nil
}

func runAwait(ctx context.Context, svc *assetsnodeskai.Service, assetID string) error {
	stepStart := time.Now()
	awaitCtx, cancel := context.WithTimeout(ctx, defaultAwaitTimeout)
	defer cancel()

	attempt := 0
	for {
		attempt++
		log.Printf("[await] polling asset_id=%q attempt=%d", assetID, attempt)
		asset, err := svc.GetAsset(awaitCtx, assetID)
		if err != nil {
			return err
		}
		log.Printf("[await] poll result attempt=%d status=%q type=%q url=%q", attempt, asset.Status, asset.Type, asset.URL)
		if !strings.EqualFold(strings.TrimSpace(asset.Status), seedanceassets.AssetStatusProcessing) {
			log.Printf("[await] asset reached terminal status elapsed=%s asset_id=%q status=%q", time.Since(stepStart).Round(time.Millisecond), asset.ID, asset.Status)
			fmt.Println("id:", asset.ID)
			fmt.Println("group_id:", asset.GroupID)
			fmt.Println("name:", asset.Name)
			fmt.Println("type:", asset.Type)
			fmt.Println("status:", asset.Status)
			fmt.Println("url:", asset.URL)
			fmt.Println("create_time:", asset.CreateTime)
			return nil
		}

		select {
		case <-awaitCtx.Done():
			return awaitCtx.Err()
		case <-time.After(defaultAwaitInterval):
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./examples/seedance_nodeskai_assets create-group [name] [description]")
	fmt.Println("  go run ./examples/seedance_nodeskai_assets upload /path/to/file group_id [name]")
	fmt.Println("    # upload first, then automatically await the returned asset_id")
	fmt.Println("  go run ./examples/seedance_nodeskai_assets get asset_id")
	fmt.Println("  go run ./examples/seedance_nodeskai_assets await asset_id")
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return wd
}

func maskClientID(clientID string) string {
	clientID = strings.TrimSpace(clientID)
	if len(clientID) <= 8 {
		return clientID
	}
	return clientID[:6] + "..." + clientID[len(clientID)-2:]
}
