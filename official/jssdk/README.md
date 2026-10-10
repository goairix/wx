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

获取 ticket 的成功响应必须包含非空 `ticket` 和大于零的 `expires_in`。无效响应会返回错误，不生成签名配置，也不会缓存为空凭据；后续调用仍可重新获取有效 ticket。
