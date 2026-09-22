package kf

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// Message 客服消息
type Message struct {
	account contracts.AccountInterface
}

// NewMessage 创建 Message 实例。
func NewMessage(account contracts.AccountInterface) *Message {
	return &Message{account: account}
}

// SyncMsg 读取消息
func (m *Message) SyncMsg(req SyncMsgRequest) (*SyncMsgResult, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/sync_msg?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result syncMsgResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.SyncMsgResult, nil
}

// SendText 发送文本消息
func (m *Message) SendText(toUser, openKfid, content string, msgid ...string) (string, error) {
	req := sendMsgRequest{
		ToUser:   toUser,
		OpenKfid: openKfid,
		MsgType:  "text",
		Text:     &TextContent{Content: content},
	}
	if len(msgid) > 0 {
		req.MsgId = msgid[0]
	}
	return m.sendMsg(req)
}

// SendImage 发送图片消息
func (m *Message) SendImage(toUser, openKfid, mediaId string, msgid ...string) (string, error) {
	req := sendMsgRequest{
		ToUser:   toUser,
		OpenKfid: openKfid,
		MsgType:  "image",
		Image:    &MediaContent{MediaId: mediaId},
	}
	if len(msgid) > 0 {
		req.MsgId = msgid[0]
	}
	return m.sendMsg(req)
}

// SendVoice 发送语音消息
func (m *Message) SendVoice(toUser, openKfid, mediaId string, msgid ...string) (string, error) {
	req := sendMsgRequest{
		ToUser:   toUser,
		OpenKfid: openKfid,
		MsgType:  "voice",
		Voice:    &MediaContent{MediaId: mediaId},
	}
	if len(msgid) > 0 {
		req.MsgId = msgid[0]
	}
	return m.sendMsg(req)
}

// SendVideo 发送视频消息
func (m *Message) SendVideo(toUser, openKfid, mediaId string, msgid ...string) (string, error) {
	req := sendMsgRequest{
		ToUser:   toUser,
		OpenKfid: openKfid,
		MsgType:  "video",
		Video:    &MediaContent{MediaId: mediaId},
	}
	if len(msgid) > 0 {
		req.MsgId = msgid[0]
	}
	return m.sendMsg(req)
}

func (m *Message) sendMsg(req sendMsgRequest) (string, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/kf/send_msg?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result sendMsgResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.MsgId, nil
}
