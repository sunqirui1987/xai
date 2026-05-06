// Seedance (NoDesk AI) digital assets example.
//
// Usage:
//
//	export NODESKAI_CLIENT_ID=ndapp_xxx
//	export NODESKAI_CLIENT_SECRET=your-secret
//	go run ./examples/seedance_nodeskai_assets upload /path/to/portrait.jpg grp_abc123 "女性正脸-01"
//	go run ./examples/seedance_nodeskai_assets get asset_789xyz
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	start := time.Now()
	clientID := os.Getenv("NODESKAI_CLIENT_ID")
	clientSecret := os.Getenv("NODESKAI_CLIENT_SECRET")
	log.Printf("[boot] seedance_nodeskai_assets started cwd=%q args=%q", mustGetwd(), os.Args)
	log.Printf("[boot] env client_id_set=%t client_secret_set=%t", clientID != "", clientSecret != "")
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
	default:
		printUsage()
		os.Exit(1)
	}

	log.Printf("[done] command=%q elapsed=%s", os.Args[1], time.Since(start).Round(time.Millisecond))
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
	return nil
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

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./examples/seedance_nodeskai_assets upload /path/to/file group_id [name]")
	fmt.Println("  go run ./examples/seedance_nodeskai_assets get asset_id")
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
