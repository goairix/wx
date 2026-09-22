package customer

import "github.com/goairix/wx/kernel/contracts"

// Customer 客户联系
type Customer struct {
	account contracts.AccountInterface
}

// NewCustomer 创建 Customer 实例。
func NewCustomer(account contracts.AccountInterface) *Customer {
	return &Customer{account: account}
}

// ExternalContact 客户管理
func (c *Customer) ExternalContact() *ExternalContact {
	return NewExternalContact(c.account)
}

// Tag 客户标签管理
func (c *Customer) Tag() *Tag {
	return NewTag(c.account)
}

// Strategy 客户联系规则组管理
func (c *Customer) Strategy() *Strategy {
	return NewStrategy(c.account)
}

// GroupChat 客户群管理
func (c *Customer) GroupChat() *GroupChat {
	return NewGroupChat(c.account)
}
