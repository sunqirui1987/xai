# Seedance Assets

本包定义 **Seedance 2.0 数字资产库** 的独立服务层能力，和 `spec/seedance` 的视频生成能力完全分开。

## 职责

- 创建素材组
- 上传素材：图片 / 视频 / 音频
- 查询素材详情

## 代码结构

| 文件 | 说明 |
|------|------|
| `types.go` | 上传请求、上传结果、素材详情结构 |
| `backend.go` | provider 需要实现的 backend 接口 |
| `service.go` | 对外暴露的 Service |

## 使用方式

```go
import (
    "context"
    seedanceassets "github.com/goplus/xai/spec/seedance_assets"
    assetsnodeskai "github.com/goplus/xai/spec/seedance_assets/provider/nodeskai"
    assetsqiniu "github.com/goplus/xai/spec/seedance_assets/provider/qiniu"
)

func main() {
    svc := assetsnodeskai.NewServiceWithClientCredentials(clientID, clientSecret)
    _, _ = svc.UploadAsset(context.Background(), &seedanceassets.UploadAssetRequest{})

    qiniuSvc := assetsqiniu.NewService(apiKey)
    _, _ = qiniuSvc.UploadAsset(context.Background(), &seedanceassets.UploadAssetRequest{
        AssetType: "image",
        URL: "https://example.com/portrait.jpg",
        Name: "年轻男人",
        Model: "bytedance/doubao-seedance-2-0-260128",
    })
}
```

## Qiniu 对接补充

- 创建素材组：`POST /v1/asset-groups`
- 创建素材审核任务：`POST /v1/assets`
- 查询素材审核状态：`GET /v1/assets/{qassetid}`
- 国内默认端点：`https://openai.qiniu.com`
- 海外端点：`https://openai.sufy.com`，通过 `assetsqiniu.WithBaseURL(assetsqiniu.OverseasBaseURL)` 使用
- 素材 URL 必须公网可访问；返回 `approved` 后可使用 `qasset://{qassetid}` 引用

## NoDesk AI 对接补充

- OAuth 取 token：`POST https://platform-api.nodesk.tech/api/v1/oauth/token`
- token 请求字段：`grant_type=client_credentials`、`client_id`、`client_secret`
- 资产库接口鉴权：`Authorization: Bearer <access_token>`
- 按对接文档，平台资产库接口还应携带 `X-External-User-Id`
- 本实现支持通过 `NODESKAI_EXTERNAL_USER_ID` 或 `NODESK_EXTERNAL_USER_ID` 自动注入该请求头
