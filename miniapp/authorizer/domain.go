package authorizer

import (
	"context"
)

// DomainClient manages server and web-view domains.
type DomainClient struct {
	api Caller
}

// Modify updates the miniapp server domains.
func (c *DomainClient) Modify(
	ctx context.Context,
	data map[string][]string,
) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	err := c.api.Post(
		ctx,
		"miniapp.authorizer.domain.modify",
		"wxa/modify_domain",
		data,
		&result,
	)
	return result, err
}

// SetWebViewDomain updates the miniapp business domains.
func (c *DomainClient) SetWebViewDomain(
	ctx context.Context,
	action string,
	domains ...string,
) error {
	return c.api.Post(
		ctx,
		"miniapp.authorizer.domain.webview",
		"wxa/setwebviewdomain",
		map[string]interface{}{
			"action":        action,
			"webviewdomain": domains,
		},
		nil,
	)
}
