package contracts

import (
	"context"

	"github.com/goairix/wx/v2/kernel/message"
	"github.com/goairix/wx/v2/kernel/message/reply"
)

// EventHandler 事件处理函数
type EventHandler func(ctx context.Context, account AccountInterface, msg *message.Message) (*reply.Reply, error)

// Middleware 中间件
type Middleware func(next EventHandler) EventHandler
