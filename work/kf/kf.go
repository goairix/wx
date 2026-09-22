package kf

import "github.com/goairix/wx/v2/kernel/contracts"

// Kf 微信客服
type Kf struct {
	account contracts.AccountInterface
}

// NewKf 创建 Kf 实例。
func NewKf(account contracts.AccountInterface) *Kf {
	return &Kf{account: account}
}

// Account 客服账号管理
func (k *Kf) Account() *Account {
	return NewAccount(k.account)
}

// Servicer 接待人员管理
func (k *Kf) Servicer() *Servicer {
	return NewServicer(k.account)
}

// Message 客服消息
func (k *Kf) Message() *Message {
	return NewMessage(k.account)
}

// ServiceState 会话管理
func (k *Kf) ServiceState() *ServiceState {
	return NewServiceState(k.account)
}
