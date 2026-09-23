package kf

import (
	"context"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides WeChat customer service APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client {
	return &Client{
		api: executor,
	}
}

func (c *Client) AddAccount(ctx context.Context, name, mediaID string) (string, error) {
	body := struct {
		Name    string `json:"name"`
		MediaID string `json:"media_id"`
	}{
		Name:    name,
		MediaID: mediaID,
	}
	var result struct {
		OpenKFID string `json:"open_kfid"`
	}
	err := c.api.Post(ctx, "work.kf.account.add", "cgi-bin/kf/account/add", body, &result)
	return result.OpenKFID, err
}

func (c *Client) DeleteAccount(ctx context.Context, openKFID string) error {
	body := struct {
		OpenKFID string `json:"open_kfid"`
	}{
		OpenKFID: openKFID,
	}
	return c.api.Post(ctx, "work.kf.account.delete", "cgi-bin/kf/account/del", body, nil)
}

func (c *Client) UpdateAccount(ctx context.Context, input UpdateAccountRequest) error {
	return c.api.Post(ctx, "work.kf.account.update", "cgi-bin/kf/account/update", input, nil)
}

func (c *Client) ListAccounts(ctx context.Context, offset, limit int) (*AccountListResult, error) {
	body := struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
	}{
		Offset: offset,
		Limit:  limit,
	}
	result := new(AccountListResult)
	err := c.api.Post(ctx, "work.kf.account.list", "cgi-bin/kf/account/list", body, result)
	return result, err
}

func (c *Client) AccountContactWay(
	ctx context.Context,
	openKFID string,
	scene string,
) (string, error) {
	body := struct {
		OpenKFID string `json:"open_kfid"`
		Scene    string `json:"scene,omitempty"`
	}{
		OpenKFID: openKFID,
		Scene:    scene,
	}
	var result struct {
		URL string `json:"url"`
	}
	err := c.api.Post(
		ctx,
		"work.kf.account.contact_way",
		"cgi-bin/kf/add_contact_way",
		body,
		&result,
	)
	return result.URL, err
}

func (c *Client) AddServicers(
	ctx context.Context,
	openKFID string,
	userIDs []string,
) ([]ServicerResult, error) {
	return c.changeServicers(ctx, "add", openKFID, userIDs)
}

func (c *Client) DeleteServicers(
	ctx context.Context,
	openKFID string,
	userIDs []string,
) ([]ServicerResult, error) {
	return c.changeServicers(ctx, "del", openKFID, userIDs)
}

func (c *Client) changeServicers(
	ctx context.Context,
	action string,
	openKFID string,
	userIDs []string,
) ([]ServicerResult, error) {
	body := struct {
		OpenKFID string   `json:"open_kfid"`
		UserIDs  []string `json:"userid_list"`
	}{
		OpenKFID: openKFID,
		UserIDs:  userIDs,
	}
	var result struct {
		Results []ServicerResult `json:"result_list"`
	}
	err := c.api.Post(
		ctx,
		"work.kf.servicer."+action,
		"cgi-bin/kf/servicer/"+action,
		body,
		&result,
	)
	return result.Results, err
}

func (c *Client) ListServicers(ctx context.Context, openKFID string) ([]ServicerInfo, error) {
	body := struct {
		OpenKFID string `json:"open_kfid"`
	}{
		OpenKFID: openKFID,
	}
	var result struct {
		Servicers []ServicerInfo `json:"servicer_list"`
	}
	err := c.api.Post(ctx, "work.kf.servicer.list", "cgi-bin/kf/servicer/list", body, &result)
	return result.Servicers, err
}

func (c *Client) ServiceState(
	ctx context.Context,
	openKFID string,
	externalUserID string,
) (*ServiceStateInfo, error) {
	body := struct {
		OpenKFID       string `json:"open_kfid"`
		ExternalUserID string `json:"external_userid"`
	}{
		OpenKFID:       openKFID,
		ExternalUserID: externalUserID,
	}
	result := new(ServiceStateInfo)
	err := c.api.Post(ctx, "work.kf.service_state.get", "cgi-bin/kf/service_state/get", body, result)
	return result, err
}

func (c *Client) Transfer(
	ctx context.Context,
	input ServiceStateTransRequest,
) (string, error) {
	var result struct {
		MessageCode string `json:"msg_code"`
	}
	err := c.api.Post(
		ctx,
		"work.kf.service_state.transfer",
		"cgi-bin/kf/service_state/trans",
		input,
		&result,
	)
	return result.MessageCode, err
}

func (c *Client) SyncMessages(
	ctx context.Context,
	input SyncMsgRequest,
) (*SyncMsgResult, error) {
	result := new(SyncMsgResult)
	err := c.api.Post(ctx, "work.kf.message.sync", "cgi-bin/kf/sync_msg", input, result)
	return result, err
}

func (c *Client) SendText(
	ctx context.Context,
	toUser string,
	openKFID string,
	content string,
	messageID string,
) (string, error) {
	body := sendMsgRequest{
		ToUser:   toUser,
		OpenKfid: openKFID,
		MsgType:  "text",
		MsgId:    messageID,
		Text: &TextContent{
			Content: content,
		},
	}
	var result struct {
		MessageID string `json:"msgid"`
	}
	err := c.api.Post(ctx, "work.kf.message.send_text", "cgi-bin/kf/send_msg", body, &result)
	return result.MessageID, err
}

func (c *Client) SendImage(
	ctx context.Context,
	toUser string,
	openKFID string,
	mediaID string,
	messageID string,
) (string, error) {
	return c.sendMedia(ctx, "image", toUser, openKFID, mediaID, messageID)
}

func (c *Client) SendVoice(
	ctx context.Context,
	toUser string,
	openKFID string,
	mediaID string,
	messageID string,
) (string, error) {
	return c.sendMedia(ctx, "voice", toUser, openKFID, mediaID, messageID)
}

func (c *Client) SendVideo(
	ctx context.Context,
	toUser string,
	openKFID string,
	mediaID string,
	messageID string,
) (string, error) {
	return c.sendMedia(ctx, "video", toUser, openKFID, mediaID, messageID)
}

func (c *Client) sendMedia(
	ctx context.Context,
	mediaType string,
	toUser string,
	openKFID string,
	mediaID string,
	messageID string,
) (string, error) {
	body := sendMsgRequest{
		ToUser:   toUser,
		OpenKfid: openKFID,
		MsgType:  mediaType,
		MsgId:    messageID,
	}
	content := &MediaContent{
		MediaId: mediaID,
	}
	switch mediaType {
	case "image":
		body.Image = content
	case "voice":
		body.Voice = content
	case "video":
		body.Video = content
	}
	var result struct {
		MessageID string `json:"msgid"`
	}
	err := c.api.Post(
		ctx,
		"work.kf.message.send_"+mediaType,
		"cgi-bin/kf/send_msg",
		body,
		&result,
	)
	return result.MessageID, err
}
