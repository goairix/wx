package message

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// Message 应用消息
type Message struct {
	account contracts.AccountInterface
}

// NewMessage 创建 Message 实例。
func NewMessage(account contracts.AccountInterface) *Message {
	return &Message{account: account}
}

// Chat 群聊会话
func (m *Message) Chat() *Chat {
	return newChat(m)
}

// Send 发送应用消息
func (m *Message) Send(opt SendOption, msg Messenger) (*SendResult, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	body := map[string]interface{}{
		"msgtype":     msg.MsgType(),
		"agentid":     opt.AgentId,
		msg.MsgType(): msg,
	}
	if opt.ToUser != "" {
		body["touser"] = opt.ToUser
	}
	if opt.ToParty != "" {
		body["toparty"] = opt.ToParty
	}
	if opt.ToTag != "" {
		body["totag"] = opt.ToTag
	}
	if opt.Safe > 0 {
		body["safe"] = opt.Safe
	}
	if opt.EnableIdTrans > 0 {
		body["enable_id_trans"] = opt.EnableIdTrans
	}
	if opt.EnableDuplicateCheck > 0 {
		body["enable_duplicate_check"] = opt.EnableDuplicateCheck
	}
	if opt.DuplicateCheckInterval > 0 {
		body["duplicate_check_interval"] = opt.DuplicateCheckInterval
	}

	apiUrl := fmt.Sprintf("cgi-bin/message/send?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, body)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result sendResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.SendResult, nil
}

// UpdateTemplateCard 更新模版卡片消息
func (m *Message) UpdateTemplateCard(req UpdateTemplateCardRequest) (*UpdateTemplateCardResult, error) {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/message/update_template_card?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result updateTemplateCardResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.UpdateTemplateCardResult, nil
}

// Recall 撤回应用消息
func (m *Message) Recall(msgId string) error {
	accessToken, err := m.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/message/recall?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, RecallRequest{MsgId: msgId})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result recallResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result.ApiError)
	}

	return nil
}
