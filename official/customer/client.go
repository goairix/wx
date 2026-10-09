// Package customer provides official account customer service APIs.
package customer

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/goairix/wx/v2/core/auth"
	"github.com/goairix/wx/v2/core/request"
	"github.com/goairix/wx/v2/official/internal/api"
	"github.com/goairix/wx/v2/official/internal/fileupload"
)

// Client provides account management, messaging and session operations.
type Client struct{ api Caller }

// NewClient constructs a client using shared transport and credentials.
func NewClient(tr request.Caller, manager *auth.Manager) *Client {
	return NewWithCaller(api.New(tr, manager))
}

// NewWithCaller constructs a client with an authenticated caller.
func NewWithCaller(caller Caller) *Client { return &Client{api: caller} }

// AddAccount adds an account, optionally for a submerchant.
func (c *Client) AddAccount(ctx context.Context, input AccountRequest) error {
	return c.api.Post(ctx, "official.customer.account.add", "customservice/kfaccount/add", input, nil)
}

// UpdateAccount changes an account's nickname.
func (c *Client) UpdateAccount(ctx context.Context, account, nickname string) error {
	return c.api.Post(ctx, "official.customer.account.update", "customservice/kfaccount/update", map[string]string{"kf_account": account, "nickname": nickname}, nil)
}

// DeleteAccount removes an account using the current POST interface.
func (c *Client) DeleteAccount(ctx context.Context, input DeleteAccountRequest) error {
	return c.api.Post(ctx, "official.customer.account.delete", "customservice/kfaccount/del", input, nil)
}

// InviteWorker invites a WeChat user to bind to an account.
func (c *Client) InviteWorker(ctx context.Context, account, weChat string) error {
	return c.api.Post(ctx, "official.customer.account.invite", "customservice/kfaccount/inviteworker", map[string]string{"kf_account": account, "invite_wx": weChat}, nil)
}

// UploadHeadImage uploads an account avatar as a multipart media file.
func (c *Client) UploadHeadImage(ctx context.Context, account, filename string, data []byte) error {
	body, header, err := fileupload.Build(filename, data, nil)
	if err != nil {
		return err
	}
	raw, _, err := c.api.Raw(ctx, "official.customer.account.upload_head_image", http.MethodPost, "customservice/kfaccount/uploadheadimg", url.Values{"kf_account": {account}}, header, body)
	if err != nil {
		return err
	}
	// Avatar responses carry only the platform envelope, but must be valid JSON.
	var result struct {
		ErrCode json.RawMessage `json:"errcode"`
	}
	return json.Unmarshal(raw, &result)
}

// ListAccounts returns all accounts. Empty businessID selects the main account.
func (c *Client) ListAccounts(ctx context.Context, businessID string) ([]AccountInfo, error) {
	var result struct {
		Accounts []AccountInfo `json:"kf_list"`
	}
	err := c.api.Get(ctx, "official.customer.account.list", "cgi-bin/customservice/getkflist", businessQuery(businessID), &result)
	return result.Accounts, err
}

// ListOnlineAccounts returns the online accounts and accepted session counts.
func (c *Client) ListOnlineAccounts(ctx context.Context, businessID string) ([]OnlineAccount, error) {
	var result struct {
		Accounts []OnlineAccount `json:"kf_online_list"`
	}
	err := c.api.Get(ctx, "official.customer.account.online", "cgi-bin/customservice/getonlinekflist", businessQuery(businessID), &result)
	return result.Accounts, err
}

func businessQuery(businessID string) url.Values {
	if businessID == "" {
		return nil
	}
	return url.Values{"business_id": {businessID}}
}
