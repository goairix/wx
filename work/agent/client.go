package agent

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides enterprise WeChat application management APIs.
type Client struct{ api Caller }

// NewClient constructs an application client using the shared executor.
func NewClient(executor *api.Client) *Client {
	return NewWithCaller(executor)
}

// NewWithCaller constructs an application client with an authenticated caller.
func NewWithCaller(caller Caller) *Client { return &Client{api: caller} }

// Get returns settings and visibility scopes for an accessible application.
func (c *Client) Get(ctx context.Context, agentID int64) (*Detail, error) {
	result := new(Detail)
	err := c.api.Get(ctx, "work.agent.get", "cgi-bin/agent/get", agentQuery(agentID), result)
	return result, err
}

// List returns the applications accessible to the current credential.
func (c *Client) List(ctx context.Context) ([]Summary, error) {
	var result struct {
		AgentList []Summary `json:"agentlist"`
	}
	err := c.api.Get(ctx, "work.agent.list", "cgi-bin/agent/list", nil, &result)
	return result.AgentList, err
}

// Set updates the current self-built application's settings. Third-party and
// provider-developed applications cannot use this endpoint.
func (c *Client) Set(ctx context.Context, input SetRequest) error {
	return c.api.Post(ctx, "work.agent.set", "cgi-bin/agent/set", input, nil)
}

// CreateMenu replaces an application's custom menu. Enterprise credentials are required.
func (c *Client) CreateMenu(ctx context.Context, agentID int64, buttons []MenuButton) error {
	return c.api.Do(ctx, "work.agent.menu.create", http.MethodPost, "cgi-bin/menu/create",
		agentQuery(agentID), Menu{Button: buttons}, nil)
}

// GetMenu returns an application's custom menu. Enterprise credentials are required.
func (c *Client) GetMenu(ctx context.Context, agentID int64) (*Menu, error) {
	result := new(Menu)
	err := c.api.Get(ctx, "work.agent.menu.get", "cgi-bin/menu/get", agentQuery(agentID), result)
	return result, err
}

// DeleteMenu deletes an application's menu without automatic retries, despite
// using GET. Enterprise credentials are required.
func (c *Client) DeleteMenu(ctx context.Context, agentID int64) error {
	return c.api.GetOnce(ctx, "work.agent.menu.delete", "cgi-bin/menu/delete", agentQuery(agentID), nil)
}

// SetWorkbenchTemplate configures the display style and enterprise defaults.
func (c *Client) SetWorkbenchTemplate(ctx context.Context, input SetWorkbenchTemplateRequest) error {
	return c.api.Post(ctx, "work.agent.workbench.template.set", "cgi-bin/agent/set_workbench_template", input, nil)
}

// GetWorkbenchTemplate returns the application's workbench template.
func (c *Client) GetWorkbenchTemplate(ctx context.Context, agentID int64) (*WorkbenchTemplate, error) {
	result := new(WorkbenchTemplate)
	err := c.api.Post(ctx, "work.agent.workbench.template.get", "cgi-bin/agent/get_workbench_template", struct {
		AgentID int64 `json:"agentid"`
	}{AgentID: agentID}, result)
	return result, err
}

// SetWorkbenchData updates workbench data for one member in the visibility scope.
func (c *Client) SetWorkbenchData(ctx context.Context, input SetWorkbenchDataRequest) error {
	return c.api.Post(ctx, "work.agent.workbench.data.set", "cgi-bin/agent/set_workbench_data", input, nil)
}

// BatchSetWorkbenchData updates identical workbench data for up to 1000 visible members.
func (c *Client) BatchSetWorkbenchData(ctx context.Context, input BatchSetWorkbenchDataRequest) error {
	return c.api.Post(ctx, "work.agent.workbench.data.batch_set", "cgi-bin/agent/batch_set_workbench_data", input, nil)
}

// GetWorkbenchData returns a member's workbench data, including enterprise
// defaults when a replacing template overrides the member's data.
func (c *Client) GetWorkbenchData(ctx context.Context, agentID int64, userID string) (*WorkbenchData, error) {
	var result struct {
		Data WorkbenchData `json:"data"`
	}
	err := c.api.Post(ctx, "work.agent.workbench.data.get", "cgi-bin/agent/get_workbench_data", struct {
		AgentID int64  `json:"agentid"`
		UserID  string `json:"userid"`
	}{AgentID: agentID, UserID: userID}, &result)
	return &result.Data, err
}

func agentQuery(agentID int64) url.Values {
	return url.Values{"agentid": {strconv.FormatInt(agentID, 10)}}
}
