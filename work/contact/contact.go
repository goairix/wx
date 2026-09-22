package contact

import "github.com/goairix/wx/kernel/contracts"

// Contact 通讯录管理
type Contact struct {
	token   string
	aesKey  string
	account contracts.AccountInterface
}

// New 创建通讯录客户端。
func New(account contracts.AccountInterface, token, aesKey string) *Contact {
	return &Contact{
		token:   token,
		aesKey:  aesKey,
		account: account,
	}
}

// User 通讯录管理-成员管理
func (b *Contact) User() *User {
	return NewUser(b.account)
}

// Department 通讯录管理-部门管理
func (b *Contact) Department() *Department {
	return NewDepartment(b.account)
}

// Tag 通讯录管理-标签管理
func (b *Contact) Tag() *Tag {
	return NewTag(b.account)
}

// Import 通讯录管理-异步导入
func (b *Contact) Import() *Import {
	return NewImport(b.account)
}

// Export 通讯录管理-异步导出
func (b *Contact) Export() *Export {
	return NewExport(b.account, b.token, b.aesKey)
}
