// Package authorizer provides enterprise third-party authorization APIs.
package authorizer

import (
	"context"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides authorized enterprise APIs.
type Client struct{ api *api.Client }

func NewClient(executor *api.Client) *Client {
	return &Client{
		api: executor,
	}
}

// PermanentCodeResult contains an authorized enterprise permanent code.
type PermanentCodeResult struct {
	PermanentCode string                 `json:"permanent_code"`
	AuthCorpInfo  map[string]interface{} `json:"auth_corp_info"`
	AuthInfo      map[string]interface{} `json:"auth_info"`
}

// PermanentCode exchanges a temporary authorization code.
func (c *Client) PermanentCode(
	ctx context.Context,
	temporaryCode string,
) (*PermanentCodeResult, error) {
	body := struct {
		AuthCode string `json:"auth_code"`
	}{
		AuthCode: temporaryCode,
	}
	result := new(PermanentCodeResult)
	err := c.api.Post(
		ctx,
		"work.authorizer.permanent_code",
		"cgi-bin/service/get_permanent_code",
		body,
		result,
	)
	return result, err
}
