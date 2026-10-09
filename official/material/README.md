# 公众号素材管理

通过 `client.Materials()` 调用，使用 `github.com/goairix/wx/v2/official/material` 中的类型。
共享根客户端的凭据、HTTP、context、日志、观测和结构化错误处理；支持开放平台代公众号调用。

## 上传

```go
uploaded, err := client.Materials().UploadTemporary(ctx, material.TypeImage, "photo.jpg", imageBytes)

permanent, err := client.Materials().Add(ctx, material.UploadRequest{
    Type:     material.TypeVideo,
    Filename: "video.mp4",
    Data:     videoBytes,
    Description: &material.VideoDescription{
        Title:        "视频标题",
        Introduction: "视频简介",
    },
})

image, err := client.Materials().UploadImage(ctx, "article.jpg", imageBytes)
// image.URL 可用于文章正文中的图片。
```

上传使用 `multipart/form-data` 的 `media` 字段，文件名只保留 basename。视频描述以
JSON 字符串写入独立的 `description` 表单字段。临时和永久文件支持 `TypeImage`、
`TypeVoice`、`TypeVideo`、`TypeThumb`；`TypeNews` 用于查询图文素材列表。
临时缩略图的返回 ID 可能位于 `uploaded.ThumbMediaID`。
文件格式、大小和有效期遵循微信规定，由微信侧最终校验。

## 获取与下载

```go
result, err := client.Materials().Get(ctx, "永久素材media_id")
// 图片/语音/缩略图：result.Data 和 result.ContentType。
// 图文：result.NewsItem。
// 视频：result.Title、result.Description 和 result.DownURL。

temporary, err := client.Materials().GetTemporary(ctx, "临时素材media_id")
// 普通文件：temporary.Data；视频元数据：temporary.VideoURL。

voice, err := client.Materials().GetHDVoice(ctx, "JSSDK返回的serverID")
```

SDK 保留视频下载 URL，不自动访问第二个地址。`Get`、`GetTemporary`、`GetHDVoice` 缓冲完整响应，
默认最大为 32 MiB，可通过根客户端注入配置了 `transport.WithMaxResponseBytes` 的 transport 调整。

大文件可直接写入 `io.Writer`：

```go
contentType, err := client.Materials().DownloadTo(ctx, "永久素材media_id", file)
contentType, err = client.Materials().DownloadTemporaryTo(ctx, "临时素材media_id", file)
contentType, err = client.Materials().DownloadHDVoiceTo(ctx, "serverID", file)
```

流式方法直接写入原始响应；图文内容或视频 URL 的成功 JSON 也会按原样写入。
媒体文件及 HTTP 成功且明确标记为 `attachment` 的附件直接流式写入，附件中的 JSON 也作为文件原样保留；
其他 JSON 响应仍需缓冲并受配置的响应上限约束。
需要结构化元数据时使用对应的 `Get` 方法。接口返回的非附件平台错误 JSON 会转换为 `*core/errors.Error`，
不写入目标。若 context 取消或写入失败，目标可能已包含部分文件内容，应由调用方清理。

## 永久素材管理

```go
counts, err := client.Materials().Count(ctx)
page, err := client.Materials().List(ctx, material.ListRequest{
    Type:   material.TypeNews,
    Offset: 0,
    Count:  20,
})
err = client.Materials().Delete(ctx, "永久素材media_id")
```

列表支持图片、语音、视频和图文，`Offset` 从 0 开始，`Count` 为 1–20。
文件素材读取 `page.Items[i].Name`、`URL`；图文素材读取 `Content.NewsItem`。

## 完整接口映射

| 方法 | HTTP | 微信接口 |
| --- | --- | --- |
| `UploadTemporary` | POST multipart | `cgi-bin/media/upload` |
| `GetTemporary` / `DownloadTemporaryTo` | GET | `cgi-bin/media/get` |
| `GetHDVoice` / `DownloadHDVoiceTo` | GET | `cgi-bin/media/get/jssdk` |
| `Add` | POST multipart | `cgi-bin/material/add_material` |
| `Get` / `DownloadTo` | POST | `cgi-bin/material/get_material` |
| `Delete` | POST | `cgi-bin/material/del_material` |
| `List` | POST | `cgi-bin/material/batchget_material` |
| `Count` | GET | `cgi-bin/material/get_materialcount` |
| `UploadImage` | POST multipart | `cgi-bin/media/uploadimg` |

覆盖[微信素材管理官方目录](https://developers.weixin.qq.com/doc/service/api/material/permanent/api_getmaterial.html)中的
临时素材与永久素材接口。图文创建、编辑使用微信草稿和发布能力，当前素材目录未列出历史
`add_news`、`update_news` 接口。
