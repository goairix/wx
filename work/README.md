# 企业微信 SDK

企业微信（WeCom）服务端 SDK 使用手册。SDK 按业务拆分为多个模块，所有模块都从同一个 `work.Work` 实例获取。

## 模块导航

| 模块 | README | 入口 | 能力 |
| --- | --- | --- | --- |
| 核心 | [work](.) | `work.New` | 账号、token、缓存、回调服务 |
| 通讯录 | [contact](contact/) | `w.Contact()` | 成员、部门、标签、异步导入导出 |
| 客户联系 | [customer](customer/) | `w.Customer()` | 外部联系人、客户标签、客户群、规则 |
| 应用消息 | [message](message/) | `w.Message()` | 应用消息、群聊、撤回 |
| 客服 | [kf](kf/) | `w.Kf()` | 客服账号、接待人员、客服消息 |
| 素材 | [media](media/) | `w.Media()` | 临时素材和永久图片 |
| 身份验证 | [auth](auth/) | `w.Auth()` | OAuth、扫码登录、TFA |
| 账号 ID | [account_id](account_id/) | `w.AccountId()` | userid/openid/open_userid 转换 |
| 小程序 | [mini_program](mini_program/) | `w.MiniProgram()` | 小程序登录 |
| HTTP | [http](http/) | `work/http` | SDK 内部 HTTP 请求封装 |

每个模块的 README 都包含职责、主要 API、参数说明和可复制的最小示例。


## 安装

```sh
go get github.com/goairix/wx
```

## 快速开始

```go
package main

import (
    "github.com/goairix/wx/work"
    "github.com/goairix/wx/support/cache"
)

func main() {
    // 创建企业微信实例
    w := work.New(
        "corp_id",       // 企业ID
        "corp_secret",   // 应用Secret
        "token",         // 回调Token
        "aes_key",       // 回调EncodingAESKey
    )

    // 可选配置
    w = work.New("corp_id", "corp_secret", "token", "aes_key",
        work.WithCache(cache.NewMemoryCache()),      // 自定义缓存
        work.WithCacheKeyPrefix("my_prefix:"),       // 缓存key前缀
    )
}
```

## 回调事件处理

### 基础用法

```go
import (
    "context"
    "log"
    "net/http"

    "github.com/goairix/wx/kernel/contracts"
    "github.com/goairix/wx/kernel/event"
    "github.com/goairix/wx/kernel/message"
    "github.com/goairix/wx/kernel/message/reply"
    "github.com/goairix/wx/work"
)

func main() {
    w := work.New("corp_id", "corp_secret", "token", "aes_key")

    svr := w.Server()

    // 注册事件处理器
    svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
        log.Printf("收到消息: MsgType=%s, Event=%s, AgentID=%s", msg.MsgType, msg.Event, msg.AgentID)
        return nil, nil
    }, event.All) // event.All 匹配所有事件

    // 启动HTTP服务
    http.HandleFunc("/work/callback", func(w http.ResponseWriter, r *http.Request) {
        svr.Serve(r, w)
    })
    http.ListenAndServe(":80", nil)
}
```

### 文本消息

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    text := msg.Text()
    log.Printf("收到文本消息: %s", text.Content)
    return reply.NewReply(reply.NewText("收到")), nil
}, event.EventType("text"))
```

### 图片/语音/视频消息

```go
// 图片
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    img := msg.Image()
    log.Printf("图片: MediaId=%s, PicUrl=%s", img.MediaId, img.PicUrl)
    return nil, nil
}, event.EventType("image"))

// 语音
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    voice := msg.Voice()
    log.Printf("语音: MediaId=%s, Format=%s", voice.MediaId, voice.Format)
    return nil, nil
}, event.EventType("voice"))

// 视频
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    video := msg.Video()
    log.Printf("视频: MediaId=%s", video.MediaId)
    return nil, nil
}, event.EventType("video"))
```

### 位置消息

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    loc := msg.Location()
    log.Printf("位置: X=%s, Y=%s, Label=%s", loc.LocationX, loc.LocationY, loc.Label)
    return nil, nil
}, event.EventType("location"))
```

### 通讯录变更事件

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.WorkContactEvent()
    switch e.ChangeType {
    // 成员变更
    case "create_user":
        log.Printf("新增成员: UserID=%s, Name=%s, Department=%s", e.UserID, e.Name, e.Department)
    case "update_user":
        log.Printf("更新成员: UserID=%s, NewUserID=%s", e.UserID, e.NewUserID)
    case "delete_user":
        log.Printf("删除成员: UserID=%s", e.UserID)

    // 部门变更
    case "create_party":
        log.Printf("新增部门: Id=%s, Name=%s, ParentId=%s", e.Id, e.Name, e.ParentId)
    case "update_party":
        log.Printf("更新部门: Id=%s, Name=%s", e.Id, e.Name)
    case "delete_party":
        log.Printf("删除部门: Id=%s", e.Id)

    // 标签变更
    case "update_tag":
        log.Printf("标签变更: TagId=%s, AddUserItems=%s, DelUserItems=%s", e.TagId, e.AddUserItems, e.DelUserItems)
    }
    return nil, nil
}, event.ChangeContact)
```

### 外部联系人变更事件

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.WorkExternalContactEvent()
    switch e.ChangeType {
    case "add_external_contact":
        log.Printf("添加外部联系人: UserID=%s, ExternalUserID=%s", e.UserID, e.ExternalUserID)
    case "edit_external_contact":
        log.Printf("编辑外部联系人: UserID=%s, ExternalUserID=%s", e.UserID, e.ExternalUserID)
    case "add_half_external_contact":
        log.Printf("外部联系人免验证添加: UserID=%s, ExternalUserID=%s", e.UserID, e.ExternalUserID)
    case "del_external_contact":
        log.Printf("删除外部联系人: UserID=%s, ExternalUserID=%s", e.UserID, e.ExternalUserID)
    case "del_follow_user":
        log.Printf("删除跟进成员: UserID=%s, ExternalUserID=%s", e.UserID, e.ExternalUserID)
    }
    return nil, nil
}, event.ChangeExternalContact)
```

### 客户群变更事件

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.WorkExternalChatEvent()
    switch e.ChangeType {
    case "create":
        log.Printf("创建客户群: ChatId=%s", e.ChatId)
    case "update":
        log.Printf("客户群变更: ChatId=%s, UpdateDetail=%s", e.ChatId, e.UpdateDetail)
    case "dismiss":
        log.Printf("解散客户群: ChatId=%s", e.ChatId)
    }
    return nil, nil
}, event.ChangeExternalChat)
```

### 模板卡片事件

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.WorkTemplateCardEvent()
    log.Printf("模板卡片: EventKey=%s, TaskId=%s, CardType=%s, ResponseCode=%s",
        e.EventKey, e.TaskId, e.CardType, e.ResponseCode)
    return nil, nil
}, event.TemplateCardEvent)
```

### 审批事件

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.WorkApprovalEvent()
    info := e.ApprovalInfo
    log.Printf("审批事件: SpNo=%s, SpName=%s, SpStatus=%d", info.SpNo, info.SpName, info.SpStatus)
    return nil, nil
}, event.SysApprovalChange)
```

### 异步任务完成事件

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    e := msg.WorkBatchJobResultEvent()
    log.Printf("异步任务完成: JobId=%s, JobType=%s, ErrCode=%d", e.BatchJob.JobId, e.BatchJob.JobType, e.BatchJob.ErrCode)
    return nil, nil
}, event.BatchJobResult)
```

### 进入应用 / 关注 / 取消关注事件

```go
svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    log.Println("用户进入应用")
    return nil, nil
}, event.EnterAgent)

svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    log.Println("用户关注")
    return nil, nil
}, event.Subscribe)

svr.On(func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
    log.Println("用户取消关注")
    return nil, nil
}, event.Unsubscribe)
```

### 中间件

```go
// 日志中间件
svr.Use(func(next contracts.EventHandler) contracts.EventHandler {
    return func(ctx context.Context, account contracts.AccountInterface, msg *message.Message) (*reply.Reply, error) {
        log.Printf("[middleware] MsgType=%s Event=%s AgentID=%s", msg.MsgType, msg.Event, msg.AgentID)
        return next(ctx, account, msg)
    }
})
```

## 通讯录管理

### 成员管理

```go
contact := w.Contact()
user := contact.User()

// 创建成员
err := user.Create(contact.CreateUserRequest{
    Userid:     "zhangsan",
    Name:       "张三",
    Mobile:     "13800138000",
    Department: []int{1, 2},
    Gender:     "1",
    Email:      "zhangsan@example.com",
})

// 获取成员
info, err := user.Get("zhangsan")

// 更新成员
err = user.Update(contact.UpdateUserRequest{
    Userid:   "zhangsan",
    Name:     "张三丰",
    Position: "工程师",
})

// 删除成员
err = user.Delete("zhangsan")

// 批量删除
err = user.BatchDelete([]string{"zhangsan", "lisi"})

// 获取部门成员（简略）
simpleList, err := user.SimpleList(1)

// 获取部门成员（详情）
detailList, err := user.DetailList(1)

// 手机号获取userid
userid, err := user.GetUseridByMobile("13800138000")

// 邮箱获取userid
userid, err = user.GetUseridByEmail("zhangsan@example.com", 1)

// userid转openid
openid, err := user.ConvertToOpenid("zhangsan")

// 邀请成员
result, err := user.Invite([]string{"zhangsan"}, []int{1}, []int{})
```

### 部门管理

```go
dept := contact.Department()

// 创建部门
deptId, err := dept.Create(contact.CreateDepartmentRequest{
    Name:     "研发部",
    Parentid: 1,
    Order:    1,
})

// 获取部门详情
info, err := dept.Get(deptId)

// 获取部门列表
list, err := dept.List(0) // 0表示获取全量

// 获取子部门ID列表
simpleList, err := dept.SimpleList(1)

// 更新部门
err = dept.Update(contact.UpdateDepartmentRequest{
    Id:   deptId,
    Name: "技术研发部",
})

// 删除部门
err = dept.Delete(deptId)
```

### 标签管理

```go
tag := contact.Tag()

// 创建标签
tagId, err := tag.Create("开发组", 0)

// 获取标签成员
detail, err := tag.Get(tagId)

// 添加标签成员
result, err := tag.AddTagUsers(tagId, []string{"zhangsan", "lisi"}, []int{1})

// 移除标签成员
result, err = tag.DelTagUsers(tagId, []string{"lisi"}, nil)

// 获取所有标签
list, err := tag.List()

// 更新标签名
err = tag.Update(tagId, "核心开发组")

// 删除标签
err = tag.Delete(tagId)
```

### 异步导入/导出

```go
// 异步导入
imp := contact.Import()

// 增量更新成员
jobId, err := imp.SyncUser("media_id", true, nil)

// 全量覆盖成员
jobId, err = imp.ReplaceUser("media_id", false, &contact.BatchCallback{
    Url:            "https://example.com/callback",
    Token:          "token",
    EncodingAesKey: "aes_key",
})

// 查询任务结果
result, err := imp.GetResult(jobId)

// 异步导出
exp := contact.Export()

jobId, err = exp.User(1000)              // 导出成员详情
jobId, err = exp.SimpleUser(1000)        // 导出成员简略信息
jobId, err = exp.Department(1000)        // 导出部门
jobId, err = exp.TagUser(1000, tagId)    // 导出标签成员

result, err := exp.GetResult(jobId)
// result.DataList[0].Url  下载链接
// result.DataList[0].Size 文件大小
// result.DataList[0].Md5  文件MD5
```

## 应用消息

### 发送消息

```go
// 发送文本消息
result, err := w.Message().Send(message.SendOption{
    ToUser:  "zhangsan|lisi",
    AgentId: 1000002,
}, &message.Text{Content: "你好"})

// 发送图片消息
result, err = w.Message().Send(message.SendOption{
    ToUser:  "zhangsan",
    AgentId: 1000002,
}, &message.Image{MediaId: "media_id"})

// 发送文件消息
result, err = w.Message().Send(message.SendOption{
    ToUser:  "zhangsan",
    AgentId: 1000002,
}, &message.File{MediaId: "media_id"})

// 发送文本卡片消息
result, err = w.Message().Send(message.SendOption{
    ToUser:  "zhangsan",
    AgentId: 1000002,
}, &message.TextCard{
    Title:       "标题",
    Description: "描述",
    Url:         "https://example.com",
    BtnTxt:      "详情",
})

// 发送Markdown消息
result, err = w.Message().Send(message.SendOption{
    ToUser:  "zhangsan",
    AgentId: 1000002,
}, &message.Markdown{
    Content: "# 标题\n> 引用\n**加粗**",
})

// 撤回消息
err = w.Message().Recall(result.MsgId)
```

所有消息类型均实现了 `Messenger` 接口，支持的类型：`Text`、`Image`、`Voice`、`Video`、`File`、`TextCard`、`News`、`MpNews`、`Markdown`、`MiniProgramNotice`、`TemplateCard`。

### 群聊消息

```go
chat := w.Message().Chat()

// 创建群聊
chatId, err := chat.Create(message.CreateChatRequest{
    Name:     "技术讨论群",
    Owner:    "zhangsan",
    UserList: []string{"zhangsan", "lisi", "wangwu"},
})

// 获取群聊信息
info, err := chat.Get(chatId)

// 修改群聊
err = chat.Update(message.UpdateChatRequest{
    ChatId:      chatId,
    Name:        "新群名",
    AddUserList: []string{"zhaoliu"},
})

// 向群聊发送文本消息（支持@群成员）
err = chat.Send(chatId, &message.Text{
    Content:       "你的快递已到\n请携带工卡前往邮件中心领取",
    MentionedList: []string{"wangqing", "@all"},
})

// 向群聊发送图片消息
err = chat.Send(chatId, &message.Image{MediaId: "media_id"})

// 向群聊发送文本卡片消息
err = chat.Send(chatId, &message.TextCard{
    Title:       "领奖通知",
    Description: "恭喜你抽中iPhone一台",
    Url:         "https://work.weixin.qq.com/",
    BtnTxt:      "更多",
})

// 发送保密消息（safe=1）
err = chat.Send(chatId, &message.Text{Content: "保密内容"}, 1)
```

## 身份验证

### OAuth2网页授权

```go
auth := w.Auth()

// 构造授权链接
url := auth.
    WithRedirectUrl("https://example.com/callback").
    WithScope("snsapi_base").
    WithState("state").
    WithAgentId("1000002").
    AuthUrl()

// 或者直接重定向
auth.Redirect(w, r)

// 通过code获取用户身份
identity, err := auth.UserFromCode("code")
// identity.UserId   企业成员userid
// identity.OpenId   非企业成员openid

// 获取用户敏感信息
detail, err := auth.GetUserDetail(identity.UserTicket)
// detail.Mobile, detail.Email, detail.Avatar
```

### 扫码登录

```go
// 构造扫码登录链接
url := auth.
    WithRedirectUrl("https://example.com/callback").
    WithState("state").
    WithAgentId("1000002").
    QrLoginUrl("login_type")
```

### 二次验证(TFA)

```go
tfa := auth.TFA()

// 获取二次验证信息
tfaInfo, err := tfa.GetTfaInfo("code")

// 通知登录成功
err = tfa.AuthSucc("userid")

// 验证TFA码
err = tfa.TfaSucc("userid", tfaInfo.TfaCode)
```

## 素材管理

```go
media := w.Media()

// 上传临时素材（image/voice/video/file）
result, err := media.Upload("image", "photo.jpg", fileData)
// result.MediaId  素材ID
// result.Type     素材类型

// 获取临时素材
data, contentType, err := media.Get("media_id")

// 获取高清语音素材
data, contentType, err = media.GetJssdk("media_id")

// 上传永久图片（返回URL）
imgResult, err := media.UploadImage("logo.png", fileData)
// imgResult.Url  图片URL

// 异步上传素材
asyncResult, err := media.AsyncUpload(media.AsyncUploadRequest{
    Scene:     "1",
    MediaType: "image",
    UploadUrl: "https://example.com/image.png",
    FileName:  "image.png",
    Md5:       "md5hash",
})
// asyncResult.JobId
```

## 账号ID转换

```go
accountId := w.AccountId()

// userid 与 openid 互转
openid, err := accountId.ConvertToOpenid("zhangsan")
userid, err := accountId.ConvertToUserid("openid")

// 批量 userid 转 openuserid
result, err := accountId.UseridToOpenuserid([]string{"zhangsan", "lisi"})

// 批量 openuserid 转 userid
result, err := accountId.OpenuseridToUserid([]string{"openuserid1"}, 1000002)

// 临时外部联系人ID转换
result, err := accountId.ConvertTmpExternalUserid([]string{"tmp_id"}, 1, 1)
```

## 企业微信小程序

```go
mp := w.MiniProgram()

// 获取session
session, err := mp.Auth().Session("js_code")
// session.CorpId
// session.UserId
// session.SessionKey
```

## 事件类型速查

| 事件常量 | 值 | 说明 |
|---------|---|------|
| `event.ChangeContact` | `change_contact` | 通讯录变更 |
| `event.BatchJobResult` | `batch_job_result` | 异步任务完成 |
| `event.EnterAgent` | `enter_agent` | 进入应用 |
| `event.Subscribe` | `subscribe` | 关注 |
| `event.Unsubscribe` | `unsubscribe` | 取消关注 |
| `event.ChangeExternalContact` | `change_external_contact` | 外部联系人变更 |
| `event.ChangeExternalChat` | `change_external_chat` | 客户群变更 |
| `event.ChangeExternalTag` | `change_external_tag` | 企业客户标签变更 |
| `event.TemplateCardEvent` | `template_card_event` | 模板卡片事件 |
| `event.TemplateCardMenu` | `template_card_menu_event` | 模板卡片菜单事件 |
| `event.SysApprovalChange` | `sys_approval_change` | 审批状态变更 |
| `event.OpenApprovalChange` | `open_approval_change` | 自建审批状态变更 |
| `event.LivingStatusChange` | `living_status_change` | 直播事件 |
| `event.MsgAuditNotify` | `msgaudit_notify` | 会话存档事件 |
| `event.ShareAgentChange` | `share_agent_change` | 共享应用事件 |
| `event.ShareChain` | `share_chain` | 上下游共享应用事件 |
| `event.Location` | `location` | 位置上报 |
| `event.Click` | `click` | 菜单点击 |
| `event.View` | `view` | 菜单跳转 |
| `event.ScancodePush` | `scancode_push` | 扫码推事件 |
| `event.ScancodeWaitMsg` | `scancode_waitmsg` | 扫码等待消息 |
| `event.PicSysPhoto` | `pic_sysphoto` | 系统拍照 |
| `event.PicPhotoOrAlbum` | `pic_photo_or_album` | 拍照或相册 |
| `event.PicWeiXin` | `pic_weixin` | 微信相册 |
| `event.LocationSelect` | `location_select` | 位置选择 |
