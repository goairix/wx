# 公众号菜单

通过 `client.Menu()` 管理公众号自定义菜单。所有网络方法均接收 `context.Context`。

```go
err := client.Menu().Create(ctx, []menu.Item{
    {Type: "view", Name: "首页", URL: "https://example.com"},
})

info, err := client.Menu().Info(ctx)
err = client.Menu().Delete(ctx)
```
