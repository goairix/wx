package message

import (
	"encoding/json"
	"fmt"

	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// Chat 群聊会话
type Chat struct {
	message *Message
}

func newChat(message *Message) *Chat {
	return &Chat{message: message}
}

// Create 创建群聊会话
func (c *Chat) Create(req CreateChatRequest) (string, error) {
	accessToken, err := c.message.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/appchat/create?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result createChatResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.ChatId, nil
}

// Update 修改群聊会话
func (c *Chat) Update(req UpdateChatRequest) error {
	accessToken, err := c.message.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/appchat/update?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// Get 获取群聊会话
func (c *Chat) Get(chatId string) (*ChatInfo, error) {
	accessToken, err := c.message.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/appchat/get?access_token=%s&chatid=%s", accessToken.AccessToken, chatId)
	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result getChatResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.ChatInfo, nil
}

// Send 应用推送消息到群聊会话
func (c *Chat) Send(chatId string, msg Messenger, safe ...int) error {
	accessToken, err := c.message.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	body := map[string]interface{}{
		"chatid":      chatId,
		"msgtype":     msg.MsgType(),
		msg.MsgType(): msg,
	}
	if len(safe) > 0 && safe[0] > 0 {
		body["safe"] = safe[0]
	}

	apiUrl := fmt.Sprintf("cgi-bin/appchat/send?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, body)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}
