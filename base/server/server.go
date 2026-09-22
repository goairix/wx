package server

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/goairix/wx/v2/kernel/contracts"
	"github.com/goairix/wx/v2/kernel/event"
	"github.com/goairix/wx/v2/kernel/message"
	"github.com/goairix/wx/v2/kernel/message/reply"
	"github.com/goairix/wx/v2/support/encryptor"
)

const (
	SuccessEmptyResponse = "success"
	EchoStr              = "echostr"
)

// EncryptMode 加密模式
type EncryptMode int

const (
	// EncryptModeAuto 自动检测 aes/明文（公众号）
	EncryptModeAuto EncryptMode = iota
	// EncryptModeAES 强制 AES 加密（企业微信）
	EncryptModeAES
)

// EchoStrMode echostr验证模式
type EchoStrMode int

const (
	// EchoStrPlain 明文直接返回（公众号）
	EchoStrPlain EchoStrMode = iota
	// EchoStrDecrypt 需要解密后返回（企业微信）
	EchoStrDecrypt
)

// ServerOption 服务端配置
type ServerOption func(*Server)

// WithEncryptMode 设置加密模式
func WithEncryptMode(mode EncryptMode) ServerOption {
	return func(s *Server) {
		s.encryptMode = mode
	}
}

// WithEchoStrMode 设置echostr验证模式
func WithEchoStrMode(mode EchoStrMode) ServerOption {
	return func(s *Server) {
		s.echoStrMode = mode
	}
}

// Server 统一服务端
type Server struct {
	mu          sync.RWMutex
	account     contracts.AccountInterface
	handlers    map[event.EventType][]contracts.EventHandler
	middleware  []contracts.Middleware
	encryptMode EncryptMode
	echoStrMode EchoStrMode
}

// New 创建服务端实例
func New(account contracts.AccountInterface, opts ...ServerOption) *Server {
	s := &Server{
		account:  account,
		handlers: make(map[event.EventType][]contracts.EventHandler),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// On 注册事件处理器，支持多个事件类型绑定同一个 handler
func (s *Server) On(handler contracts.EventHandler, eventTypes ...event.EventType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, et := range eventTypes {
		s.handlers[et] = append(s.handlers[et], handler)
	}
}

// Use 注册全局中间件
func (s *Server) Use(middlewares ...contracts.Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.middleware = append(s.middleware, middlewares...)
}

// Dispatch 直接调度事件处理（跳过路由匹配）
func (s *Server) Dispatch(
	request *http.Request,
	writer http.ResponseWriter,
	handler contracts.EventHandler,
) {
	s.serveWith(request, writer, func(ctx context.Context, msg *message.Message) (*reply.Reply, error) {
		wrapped := s.applyMiddleware(handler)
		return wrapped(ctx, s.account, msg)
	})
}

// Serve 处理微信回调请求
func (s *Server) Serve(request *http.Request, writer http.ResponseWriter) {
	s.serveWith(request, writer, func(ctx context.Context, msg *message.Message) (*reply.Reply, error) {
		handlers := s.matchHandlers(msg)
		for _, handler := range handlers {
			wrapped := s.applyMiddleware(handler)
			r, err := wrapped(ctx, s.account, msg)
			if err != nil {
				return nil, err
			}
			if r != nil {
				return r, nil
			}
		}
		return nil, nil
	})
}

// serveWith 统一请求处理流程
func (s *Server) serveWith(
	request *http.Request,
	writer http.ResponseWriter,
	exec func(ctx context.Context, msg *message.Message) (*reply.Reply, error),
) {
	params, encrypt := s.parseRequest(request)

	// 公众号使用signature验签，企业微信使用msg_signature在后续流程中验签
	if s.encryptMode != EncryptModeAES {
		if !encrypt.ValidSignature(params.timestamp, params.nonce, params.signature) {
			log.Println("signature is invalid")
			return
		}
	}

	if e := request.FormValue(EchoStr); e != "" {
		s.handleEchoStr(writer, encrypt, params, e)
		return
	}

	if request.Method == http.MethodPost {
		messageBody := s.decryptMessage(request, encrypt, params)
		if messageBody == nil {
			_, _ = writer.Write([]byte(SuccessEmptyResponse))
			return
		}

		ctx := request.Context()
		r, err := exec(ctx, messageBody)
		if err != nil {
			log.Printf("handler error: %+v", err)
			_, _ = writer.Write([]byte(SuccessEmptyResponse))
			return
		}

		if r != nil {
			s.writeReply(writer, encrypt, params, messageBody, r)
			return
		}
	}

	_, _ = writer.Write([]byte(SuccessEmptyResponse))
}

// requestParams 请求参数
type requestParams struct {
	timestamp    string
	nonce        string
	signature    string
	encryptType  string
	msgSignature string
}

// parseRequest 解析请求参数并创建加密器
func (s *Server) parseRequest(request *http.Request) (*requestParams, *encryptor.Encryptor) {
	_ = request.ParseForm()

	params := &requestParams{
		timestamp:    strings.Join(request.Form["timestamp"], ""),
		nonce:        strings.Join(request.Form["nonce"], ""),
		signature:    strings.Join(request.Form["signature"], ""),
		encryptType:  strings.Join(request.Form["encrypt_type"], ""),
		msgSignature: strings.Join(request.Form["msg_signature"], ""),
	}

	var appId = s.account.AppId()
	if s.account.IsOpenPlatform() {
		appId = s.account.ComponentAppId()
	}

	encrypt := encryptor.NewEncryptor(appId, s.account.Token(), s.account.AesKey())
	return params, encrypt
}

// handleEchoStr 处理echostr验证
func (s *Server) handleEchoStr(writer http.ResponseWriter, encrypt *encryptor.Encryptor, params *requestParams, echoStr string) {
	switch s.echoStrMode {
	case EchoStrDecrypt:
		// 企业微信：验证msg_signature并解密echostr
		if !encrypt.ValidMsgSignature(params.timestamp, params.nonce, echoStr, params.msgSignature) {
			log.Println("msg_signature is invalid")
			return
		}
		cipherData, err := base64.StdEncoding.DecodeString(echoStr)
		if err != nil {
			log.Println("Decode base64 error:", err)
			return
		}
		plainData, err := encrypt.AesDecrypt(cipherData)
		if err != nil {
			log.Println("Aes decrypt error:", err)
			return
		}
		// 从明文中提取msg内容（明文格式: random + msg_len + msg + receiveid）
		msg, err := encrypt.ExtractMsg(plainData)
		if err != nil {
			log.Println("Extract msg error:", err)
			return
		}
		_, _ = writer.Write(msg)
	default:
		// 公众号：直接返回明文echostr
		_, _ = writer.Write([]byte(echoStr))
	}
}

// isAESEncrypt 判断是否使用AES加密
func (s *Server) isAESEncrypt(params *requestParams) bool {
	if s.encryptMode == EncryptModeAES {
		return true
	}
	return params.encryptType == "aes"
}

// decryptMessage 解密消息体
func (s *Server) decryptMessage(request *http.Request, encrypt *encryptor.Encryptor, params *requestParams) *message.Message {
	var messageBody *message.Message

	if s.isAESEncrypt(params) {
		encryptRequestBody, err := encrypt.ParseEncryptBody(request)
		if err != nil {
			log.Printf("parse encrypt error: %+v", err)
			return nil
		}

		if !encrypt.ValidMsgSignature(params.timestamp, params.nonce, encryptRequestBody.Encrypt, params.msgSignature) {
			log.Println("msg_signature is invalid")
			return nil
		}

		cipherData, err := base64.StdEncoding.DecodeString(encryptRequestBody.Encrypt)
		if err != nil {
			log.Println("Decode base64 error:", err)
			return nil
		}

		plainData, err := encrypt.AesDecrypt(cipherData)
		if err != nil {
			log.Println("Aes decrypt error:", err)
			return nil
		}

		messageBody, _ = encrypt.ParseEncryptTextBody(plainData)
	} else {
		messageBody, _ = encrypt.ParseTextBody(request)
	}

	if messageBody == nil {
		return nil
	}

	if messageBody.MsgType == "" && messageBody.InfoType != "" {
		messageBody.MsgType = messageBody.InfoType
	}

	return messageBody
}

// matchHandlers 根据消息类型匹配handler
func (s *Server) matchHandlers(msg *message.Message) []contracts.EventHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if msg.MsgType == "event" {
		// 优先级：精确事件类型 > event通配 > 全局通配
		if h, ok := s.handlers[event.EventType(strings.ToLower(msg.Event))]; ok {
			return h
		}
		if h, ok := s.handlers[event.Event]; ok {
			return h
		}
		if h, ok := s.handlers[event.All]; ok {
			return h
		}
	} else {
		// 优先级：精确消息类型 > 全局通配
		if h, ok := s.handlers[event.EventType(msg.MsgType)]; ok {
			return h
		}
		if h, ok := s.handlers[event.All]; ok {
			return h
		}
	}

	return nil
}

// applyMiddleware 应用中间件
func (s *Server) applyMiddleware(handler contracts.EventHandler) contracts.EventHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()

	wrapped := handler
	for i := len(s.middleware) - 1; i >= 0; i-- {
		wrapped = s.middleware[i](wrapped)
	}
	return wrapped
}

// writeReply 写入回复
func (s *Server) writeReply(writer http.ResponseWriter, encrypt *encryptor.Encryptor, params *requestParams, msg *message.Message, r *reply.Reply) {
	replier := r.Replier()
	xmlBody := replier.BuildXml(msg.ToUserName, msg.FromUserName)

	var replyBody []byte
	if s.isAESEncrypt(params) {
		replyBody, _ = encrypt.MakeEncryptBody(xmlBody, params.timestamp, params.nonce)
	} else {
		replyBody = xmlBody
	}

	writer.Header().Set("Content-Type", replier.ContentType())
	_, _ = writer.Write(replyBody)
}
