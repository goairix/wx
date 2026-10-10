# 企业微信应用管理

通过 `client.Agent()` 访问应用设置、菜单和工作台展示。领域客户端复用根客户端的
transport、凭证缓存、日志、观测和错误处理，也可用 `agent.NewWithCaller` 注入调用器。
所有网络方法接收 `context.Context`，所有应用 ID 都显式传入。

## 接口列表

| 方法 | HTTP 接口 | 官方文档 |
| --- | --- | --- |
| `Get(ctx, agentID)` | GET `cgi-bin/agent/get` | [获取应用](https://developer.work.weixin.qq.com/document/path/90227) |
| `List(ctx)` | GET `cgi-bin/agent/list` | [获取应用列表](https://developer.work.weixin.qq.com/document/path/90227) |
| `Set(ctx, input)` | POST `cgi-bin/agent/set` | [设置应用](https://developer.work.weixin.qq.com/document/path/90228) |
| `CreateMenu(ctx, agentID, buttons)` | POST `cgi-bin/menu/create` | [创建菜单](https://developer.work.weixin.qq.com/document/path/90231) |
| `GetMenu(ctx, agentID)` | GET `cgi-bin/menu/get` | [获取菜单](https://developer.work.weixin.qq.com/document/path/90232) |
| `DeleteMenu(ctx, agentID)` | GET `cgi-bin/menu/delete` | [删除菜单](https://developer.work.weixin.qq.com/document/path/90233) |
| `SetWorkbenchTemplate(ctx, input)` | POST `cgi-bin/agent/set_workbench_template` | [工作台展示](https://developer.work.weixin.qq.com/document/path/92535) |
| `GetWorkbenchTemplate(ctx, agentID)` | POST `cgi-bin/agent/get_workbench_template` | 同上 |
| `SetWorkbenchData(ctx, input)` | POST `cgi-bin/agent/set_workbench_data` | 同上 |
| `BatchSetWorkbenchData(ctx, input)` | POST `cgi-bin/agent/batch_set_workbench_data` | 同上 |
| `GetWorkbenchData(ctx, agentID, userID)` | POST `cgi-bin/agent/get_workbench_data` | 同上 |

菜单请求的 `agentid` 位于 query 中；工作台请求的 `agentid` 位于 JSON body 中。
删除菜单具有写入副作用，SDK 不自动重试；POST 请求也遵循现有 transport 的默认禁止重试策略。

## 应用查询与设置

```go
import "github.com/goairix/wx/v2/work/agent"

detail, err := client.Agent().Get(ctx, 1000005)
applications, err := client.Agent().List(ctx)

name, homeURL := "HR助手", "https://example.com/hr"
disabled := 0
err = client.Agent().Set(ctx, agent.SetRequest{
    AgentID:       1000005,
    Name:          &name,
    HomeURL:       &homeURL,
    IsReportEnter: &disabled,
})
```

`Detail` 包含应用名称、头像、可见人员/部门/标签、停用状态、可信域名、事件上报开关、
主页以及代开发发布状态。列表只包含 `Summary`。
`SetRequest` 的可选字段使用指针：nil 省略；指向零值或空字符串时仍会发送该值，
由企业微信决定是否允许。`LogoMediaID` 使用素材上传获得的 media ID。

## 自定义菜单

```go
err = client.Agent().CreateMenu(ctx, 1000005, []agent.MenuButton{
    {Type: "click", Name: "审批", Key: "approve"},
    {Name: "服务", SubButton: []agent.MenuButton{
        {Type: "view", Name: "主页", URL: "https://example.com"},
        {Type: "view_miniprogram", Name: "小程序", AppID: "wx-app-id", PagePath: "pages/index"},
    }},
})
menu, err := client.Agent().GetMenu(ctx, 1000005)
err = client.Agent().DeleteMenu(ctx, 1000005)
```

`MenuButton` 同样支持扫码、拍照、相册及位置选择等按钮类型；对应类型需要填写 `Key`。
一级菜单 1–3 个，二级菜单 1–5 个。名称长度、跳转地址和小程序绑定等限制由平台校验。

## 工作台展示

先在企业微信管理后台启用应用的自定义展示，再配置模板。支持 `keydata`、`image`、
`list`、`webview` 四种类型，分别使用对应的指针字段；`normal` 仅用于取消自定义模板。

```go
data := agent.WorkbenchData{
    Type: agent.WorkbenchKeyDataType,
    KeyData: &agent.WorkbenchKeyData{Items: []agent.WorkbenchKeyDataItem{
        {Key: "待审批", Data: "2", JumpURL: "https://example.com/approvals"},
    }},
}
err = client.Agent().SetWorkbenchTemplate(ctx, agent.SetWorkbenchTemplateRequest{
    AgentID: 1000005,
    WorkbenchTemplate: agent.WorkbenchTemplate{
        WorkbenchData: data,
        ReplaceUserData: false,
    },
})
template, err := client.Agent().GetWorkbenchTemplate(ctx, 1000005)

err = client.Agent().SetWorkbenchData(ctx, agent.SetWorkbenchDataRequest{
    AgentID: 1000005,
    UserID: "zhangsan",
    WorkbenchData: data,
})
err = client.Agent().BatchSetWorkbenchData(ctx, agent.BatchSetWorkbenchDataRequest{
    AgentID: 1000005,
    UserIDList: []string{"zhangsan", "lisi"},
    Data: data,
})
userData, err := client.Agent().GetWorkbenchData(ctx, 1000005, "zhangsan")
```

`ReplaceUserData: true` 会覆盖企业所有用户已有的数据。
模板和单用户设置的展示字段位于请求顶层；批量设置位于 `data` 对象内，SDK 自动匹配这些结构。
用户数据查询会解包响应的 `data`，模板查询返回展示类型、默认数据与覆盖标志。

关键数据最多 4 项，列表最多 3 项；批量设置最多 1000 个用户，用户须在应用可见范围内。
单用户设置限每用户每应用 10 次/分钟，批量设置限每应用 100000 人次/分钟。
网页主页使用 `JumpURL`，小程序主页使用 `PagePath`，应按应用主页类型选择。
webview 可配置 `Height`、`HideTitle` 和 `EnableWebviewClick`；开启页面内链接跳转会使 `JumpURL` 失效。

## 权限与回调

查询接口仅返回凭证可访问的应用。设置应用仅企业可调用，第三方及代开发自建应用不可调用；
菜单接口仅企业可调用，第三方不可调用。工作台接口针对当前凭证对应的应用。
SDK 保留平台错误码和请求信息，调用失败时使用 `errors.As` 提取 `*core/errors.Error`。

管理员切换自定义展示时，`client.Webhook()` 解析 `switch_workbench_mode` 回调。
通过 `event.WorkbenchModeSwitch()` 读取 `AgentID` 和 `Mode`（1 启用、0 停用）。
