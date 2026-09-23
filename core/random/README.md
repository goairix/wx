# 随机字符串

`random.String` 使用 `crypto/rand` 生成 URL 安全随机字符串，供回调 nonce 和签名场景使用。

```go
nonce, err := random.String(16)
```
