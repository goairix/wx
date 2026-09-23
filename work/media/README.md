# media：素材管理

`media` 支持临时素材上传下载、高清语音下载、永久图片上传和异步素材上传。

```go
result, err := client.Media().Upload(
    ctx,
    "image",
    "avatar.jpg",
    fileData,
)
if err != nil {
    return err
}

data, contentType, err := client.Media().Download(ctx, result.MediaId)
```

`Download` 和 `GetJSSDK` 返回完整字节切片，适合较小素材。下载较大素材时使用
`DownloadTo` 直接写入文件或其他 `io.Writer`：

```go
file, err := os.Create("media.bin")
if err != nil {
    return err
}
defer file.Close()

contentType, err := client.Media().DownloadTo(ctx, mediaID, file)
```

临时素材的 `mediaType` 支持企业微信接口规定的 `image`、`voice`、`video`、`file`。`AsyncUpload` 只提交远程资源描述并返回任务 ID，任务结果由企业微信异步处理。
