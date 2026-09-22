# contact：通讯录管理

`contact` 管理企业微信成员、部门、标签，并提供通讯录异步导入导出。通过 `w.Contact()` 获取入口。

## 成员、部门和标签

```go
contact := w.Contact()
user := contact.User()
err := user.Create(contact.CreateUserRequest{
    Userid: "zhangsan", Name: "张三", Mobile: "13800138000", Department: []int{1},
})
if err != nil { return err }
info, err := user.Get("zhangsan")
if err != nil { return err }

partyID, err := contact.Department().Create(contact.CreateDepartmentRequest{Name: "研发部", Parentid: 1})
if err != nil { return err }
_, err = contact.Tag().Create("核心成员", 0)
_ = partyID
return err
```

常用方法：`User.Create/Get/Update/Delete`、`Department.Create/List/Get`、`Tag.Create/Get/List`。
列表接口的部门 ID 为 `0` 时表示企业全量（以企业微信接口规则为准）。

## 异步导入导出

```go
jobID, err := contact.Import().SyncUser("media_id", true, nil)
if err != nil { return err }
result, err := contact.Import().GetResult(jobID)
if err != nil { return err }
_ = result

exportJobID, err := contact.Export().User(1000)
if err != nil { return err }
files, err := contact.Export().GetResult(exportJobID)
if err != nil { return err }
for _, file := range files.DataList { fmt.Println(file.Url, file.Md5) }
```

导入和导出是异步任务，提交成功只代表拿到 `jobID`；请按企业微信任务状态轮询 `GetResult`。
