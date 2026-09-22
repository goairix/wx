package authorizer

import (
	"github.com/goairix/wx/v2/base/open"
	"github.com/goairix/wx/v2/kernel/contracts"
)

// Authorizer 小程序授权开放平台后的相关接口
// 此类接口只能由授权到开放平台的小程序调用或直接由开放平台调用
type Authorizer struct {
	account contracts.AccountInterface
}

func New(account contracts.AccountInterface) *Authorizer {
	return &Authorizer{account: account}
}

// Account 小程序基础信息
func (authorizer *Authorizer) Account() *Account {
	return NewAccount(authorizer.account)
}

// Categories 小程序类目管理
func (authorizer *Authorizer) Categories() *Categories {
	return NewCategories(authorizer.account)
}

// Domain 小程序域名配置
func (authorizer *Authorizer) Domain() *Domain {
	return NewDomain(authorizer.account)
}

// Open 小程序绑定开放平台
func (authorizer *Authorizer) Open() *open.Open {
	return open.New(authorizer.account)
}

// Tester 小程序成员管理
func (authorizer *Authorizer) Tester() *Tester {
	return NewTester(authorizer.account)
}
