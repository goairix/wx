package contracts

// Caller 是领域模块调用健康卡平台的最小接口。
// 根 healthcard.Client 实现该接口，领域包因此可以独立使用和测试。
type Caller interface {
	Call(path string, req interface{}, result interface{}) error
	CallWithRelated(path string, req interface{}, result interface{}, relateOpenID string) error
}
