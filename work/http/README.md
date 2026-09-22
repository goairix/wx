# http：企业微信 HTTP 封装

该包是 `work` 内部使用的轻量 HTTP 层，也可用于扩展 SDK API。URI 传入相对企业微信 API 路径（例如 `cgi-bin/gettoken?...`），基础地址由包内部配置。

## 常用方法

| 方法 | 说明 |
| --- | --- |
| `Get` | GET 请求并返回响应体 |
| `GetWithRespContentType` | GET，同时返回响应 `Content-Type` |
| `Post` | 自定义 Content-Type 的 POST |
| `PostJSON` | JSON 编码后 POST |
| `PostJSONWithRespContentType` | JSON POST，同时返回响应类型 |
| `Upload` | multipart 文件上传 |

```go
body, err := http.PostJSON("cgi-bin/example", map[string]string{"name": "demo"})
if err != nil { return err }
fmt.Println(string(body))
```

调用方负责解析业务响应；HTTP 层只返回传输错误和响应内容。
