package oauth

import "github.com/goairix/wx/v2/support/lock"

type Option func(auth *OAuth)

// WithLocker 设置锁
func WithLocker(locker lock.Locker) Option {
	return func(o *OAuth) {
		o.locker = locker
	}
}
