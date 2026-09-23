// Package authorizer provides enterprise third-party authorization APIs.
package authorizer

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// Client provides authorized enterprise APIs.
type Client struct {
	transport request.Caller
}

func NewClient(caller request.Caller) *Client {
	return &Client{
		transport: caller,
	}
}

// NewWithTransport constructs an authorizer client from a shared transport.
func NewWithTransport(transportClient request.Caller) *Client {
	return &Client{
		transport: transportClient,
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
	suiteAccessToken string,
	temporaryCode string,
) (*PermanentCodeResult, error) {
	body := struct {
		AuthCode string `json:"auth_code"`
	}{
		AuthCode: temporaryCode,
	}
	result := new(PermanentCodeResult)
	meta := new(request.ResponseMeta)
	var response struct {
		*PermanentCodeResult
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	response.PermanentCodeResult = result
	err := c.transport.Do(ctx, request.Request{
		Operation: "work.authorizer.permanent_code",
		Platform:  "work",
		Method:    http.MethodPost,
		Path:      "cgi-bin/service/get_permanent_code",
		Query: url.Values{
			"suite_access_token": []string{suiteAccessToken},
		},
		Body:   body,
		Result: &response,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	if response.ErrCode != 0 {
		return nil, &wxerrors.Error{
			Platform:   "work",
			Operation:  "work.authorizer.permanent_code",
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(response.ErrCode),
			Message:    response.ErrMsg,
			RequestID:  meta.RequestID,
		}
	}
	return result, err
}
