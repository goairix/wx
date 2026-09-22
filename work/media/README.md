# media：素材管理

`media` 支持临时素材上传下载、高清语音下载、永久图片上传和异步素材上传。

```go
media := w.Media()
result, err := media.Upload("image", "avatar.jpg", fileData)
if err != nil { return err }
data, contentType, err := media.Get(result.MediaId)
if err != nil { return err }
fmt.Println(len(data), contentType)

image, err := media.UploadImage("logo.png", fileData)
if err != nil { return err }
fmt.Println(image.Url)
```

临时素材的 `mediaType` 支持企业微信接口规定的 `image`、`voice`、`video`、`file`。`AsyncUpload` 只提交远程资源描述并返回任务 ID，任务结果由企业微信异步处理。
