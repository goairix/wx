# 公众号 JS SDK

通过 `client.JSSDK()` 获取 `jsapi_ticket` 并为当前页面地址生成签名配置。

```go
config, err := client.JSSDK().BuildConfig(
    ctx,
    "https://example.com/page",
    []string{"scanQRCode"},
    false,
    false,
)
```

页面地址必须与浏览器实际用于签名的完整地址一致。
