// Package authorizer manages capabilities available to an authorized miniapp.
package authorizer

import "github.com/goairix/wx/v2/miniapp/internal/api"

// Client groups authorized miniapp account operations.
type Client struct {
	account    *AccountClient
	categories *CategoryClient
	domain     *DomainClient
	open       *OpenClient
	tester     *TesterClient
}

// NewClient constructs an authorized miniapp domain client.
func NewClient(executor *api.Client) *Client {
	return NewWithCaller(executor)
}

// NewWithCaller constructs a domain client with an authenticated caller.
func NewWithCaller(executor Caller) *Client {
	return &Client{
		account:    &AccountClient{api: executor},
		categories: &CategoryClient{api: executor},
		domain:     &DomainClient{api: executor},
		open:       &OpenClient{api: executor},
		tester:     &TesterClient{api: executor},
	}
}

// Account returns basic account information operations.
func (c *Client) Account() *AccountClient {
	return c.account
}

// Categories returns miniapp category operations.
func (c *Client) Categories() *CategoryClient {
	return c.categories
}

// Domain returns server and web-view domain operations.
func (c *Client) Domain() *DomainClient {
	return c.domain
}

// Open returns Open Platform account binding operations.
func (c *Client) Open() *OpenClient {
	return c.open
}

// Tester returns experience member operations.
func (c *Client) Tester() *TesterClient {
	return c.tester
}
