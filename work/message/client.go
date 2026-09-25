package message

import (
	"context"
	"net/url"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides application and group message APIs.
type Client struct {
	api     Caller
	agentID int64
}

func NewClient(executor *api.Client, agentID int64) *Client {
	return NewWithCaller(executor, agentID)
}

// NewWithCaller constructs a domain client with an authenticated caller.
func NewWithCaller(executor Caller, agentID int64) *Client {
	return &Client{
		api:     executor,
		agentID: agentID,
	}
}

// Send sends an application message.
func (c *Client) Send(
	ctx context.Context,
	options SendOption,
	message Messenger,
) (*SendResult, error) {
	if options.AgentId == 0 {
		options.AgentId = c.agentID
	}
	body := map[string]interface{}{
		"msgtype":         message.MsgType(),
		"agentid":         options.AgentId,
		message.MsgType(): message,
	}
	addRecipients(body, options)
	result := new(SendResult)
	err := c.api.Post(ctx, "work.message.send", "cgi-bin/message/send", body, result)
	return result, err
}

func addRecipients(body map[string]interface{}, options SendOption) {
	if options.ToUser != "" {
		body["touser"] = options.ToUser
	}
	if options.ToParty != "" {
		body["toparty"] = options.ToParty
	}
	if options.ToTag != "" {
		body["totag"] = options.ToTag
	}
	if options.Safe != 0 {
		body["safe"] = options.Safe
	}
	if options.EnableIdTrans != 0 {
		body["enable_id_trans"] = options.EnableIdTrans
	}
	if options.EnableDuplicateCheck != 0 {
		body["enable_duplicate_check"] = options.EnableDuplicateCheck
	}
	if options.DuplicateCheckInterval != 0 {
		body["duplicate_check_interval"] = options.DuplicateCheckInterval
	}
}

// UpdateTemplateCard updates an application template card.
func (c *Client) UpdateTemplateCard(
	ctx context.Context,
	input UpdateTemplateCardRequest,
) (*UpdateTemplateCardResult, error) {
	if input.AgentId == 0 {
		input.AgentId = c.agentID
	}
	result := new(UpdateTemplateCardResult)
	err := c.api.Post(
		ctx,
		"work.message.update_template_card",
		"cgi-bin/message/update_template_card",
		input,
		result,
	)
	return result, err
}

// Recall recalls an application message.
func (c *Client) Recall(ctx context.Context, messageID string) error {
	body := RecallRequest{MsgId: messageID}
	return c.api.Post(ctx, "work.message.recall", "cgi-bin/message/recall", body, nil)
}

// CreateChat creates an application group chat.
func (c *Client) CreateChat(ctx context.Context, input CreateChatRequest) (string, error) {
	var result struct {
		ChatID string `json:"chatid"`
	}
	err := c.api.Post(ctx, "work.message.chat.create", "cgi-bin/appchat/create", input, &result)
	return result.ChatID, err
}

// UpdateChat updates an application group chat.
func (c *Client) UpdateChat(ctx context.Context, input UpdateChatRequest) error {
	return c.api.Post(ctx, "work.message.chat.update", "cgi-bin/appchat/update", input, nil)
}

// GetChat returns an application group chat.
func (c *Client) GetChat(ctx context.Context, chatID string) (*ChatInfo, error) {
	var result struct {
		Chat ChatInfo `json:"chat_info"`
	}
	err := c.api.Get(
		ctx,
		"work.message.chat.get",
		"cgi-bin/appchat/get",
		url.Values{"chatid": []string{chatID}},
		&result,
	)
	return &result.Chat, err
}

// SendChat sends a message to an application group chat.
func (c *Client) SendChat(
	ctx context.Context,
	chatID string,
	message Messenger,
	safe bool,
) error {
	body := map[string]interface{}{
		"chatid":          chatID,
		"msgtype":         message.MsgType(),
		message.MsgType(): message,
	}
	if safe {
		body["safe"] = 1
	}
	return c.api.Post(ctx, "work.message.chat.send", "cgi-bin/appchat/send", body, nil)
}
