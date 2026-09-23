# 公众号带参数二维码

通过 `client.QRCode()` 创建临时或永久二维码。

```go
ticket, err := client.QRCode().Temporary(
    ctx,
    qrcode.WithStrScene("campaign-2026"),
    300,
)
imageURL := client.QRCode().URL(ticket.Ticket)
```
