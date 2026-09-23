package contact

import (
	"context"
	"encoding/base64"
	"net/url"
	"strconv"

	"github.com/goairix/wx/v2/work/internal/api"
)

// Client provides enterprise contact APIs.
type Client struct {
	users       *UserClient
	departments *DepartmentClient
	tags        *TagClient
	batch       *BatchClient
}

// NewClient constructs a contact client.
func NewClient(executor *api.Client, encodingAESKey ...string) *Client {
	aesKey := ""
	if len(encodingAESKey) > 0 {
		aesKey = encodingAESKey[0]
	}
	return &Client{
		users:       &UserClient{api: executor},
		departments: &DepartmentClient{api: executor},
		tags:        &TagClient{api: executor},
		batch: &BatchClient{
			api:            executor,
			encodingAESKey: aesKey,
		},
	}
}

// Users returns member management APIs.
func (c *Client) Users() *UserClient {
	return c.users
}

// Departments returns department management APIs.
func (c *Client) Departments() *DepartmentClient {
	return c.departments
}

// Tags returns contact tag APIs.
func (c *Client) Tags() *TagClient {
	return c.tags
}

// Batch returns contact import and export APIs.
func (c *Client) Batch() *BatchClient {
	return c.batch
}

// UserClient manages enterprise members.
type UserClient struct {
	api *api.Client
}

func (c *UserClient) Create(ctx context.Context, input CreateUserRequest) error {
	return c.api.Post(ctx, "work.contact.user.create", "cgi-bin/user/create", input, nil)
}

func (c *UserClient) Get(ctx context.Context, userID string) (*UserInfo, error) {
	result := new(UserInfo)
	err := c.api.Get(
		ctx,
		"work.contact.user.get",
		"cgi-bin/user/get",
		url.Values{"userid": []string{userID}},
		result,
	)
	return result, err
}

func (c *UserClient) Update(ctx context.Context, input UpdateUserRequest) error {
	return c.api.Post(ctx, "work.contact.user.update", "cgi-bin/user/update", input, nil)
}

func (c *UserClient) Delete(ctx context.Context, userID string) error {
	return c.api.Get(
		ctx,
		"work.contact.user.delete",
		"cgi-bin/user/delete",
		url.Values{"userid": []string{userID}},
		nil,
	)
}

func (c *UserClient) BatchDelete(ctx context.Context, userIDs []string) error {
	body := struct {
		UserIDs []string `json:"useridlist"`
	}{
		UserIDs: userIDs,
	}
	return c.api.Post(ctx, "work.contact.user.batch_delete", "cgi-bin/user/batchdelete", body, nil)
}

// DepartmentClient manages enterprise departments.
type DepartmentClient struct {
	api *api.Client
}

func (c *DepartmentClient) Create(ctx context.Context, input CreateDepartmentRequest) (int, error) {
	var result struct {
		ID int `json:"id"`
	}
	err := c.api.Post(ctx, "work.contact.department.create", "cgi-bin/department/create", input, &result)
	return result.ID, err
}

func (c *DepartmentClient) Update(ctx context.Context, input UpdateDepartmentRequest) error {
	return c.api.Post(ctx, "work.contact.department.update", "cgi-bin/department/update", input, nil)
}

func (c *DepartmentClient) Delete(ctx context.Context, id int) error {
	return c.api.Get(
		ctx,
		"work.contact.department.delete",
		"cgi-bin/department/delete",
		url.Values{"id": []string{strconv.Itoa(id)}},
		nil,
	)
}

func (c *DepartmentClient) List(ctx context.Context, id int) ([]DepartmentInfo, error) {
	query := make(url.Values)
	if id > 0 {
		query.Set("id", strconv.Itoa(id))
	}
	var result struct {
		Departments []DepartmentInfo `json:"department"`
	}
	err := c.api.Get(ctx, "work.contact.department.list", "cgi-bin/department/list", query, &result)
	return result.Departments, err
}

// TagClient manages enterprise contact tags.
type TagClient struct {
	api *api.Client
}

func (c *TagClient) Create(ctx context.Context, name string, id int) (int, error) {
	body := struct {
		Name string `json:"tagname"`
		ID   int    `json:"tagid,omitempty"`
	}{
		Name: name,
		ID:   id,
	}
	var result struct {
		ID int `json:"tagid"`
	}
	err := c.api.Post(ctx, "work.contact.tag.create", "cgi-bin/tag/create", body, &result)
	return result.ID, err
}

func (c *TagClient) Update(ctx context.Context, id int, name string) error {
	body := struct {
		ID   int    `json:"tagid"`
		Name string `json:"tagname"`
	}{
		ID:   id,
		Name: name,
	}
	return c.api.Post(ctx, "work.contact.tag.update", "cgi-bin/tag/update", body, nil)
}

func (c *TagClient) Delete(ctx context.Context, id int) error {
	return c.api.Get(
		ctx,
		"work.contact.tag.delete",
		"cgi-bin/tag/delete",
		url.Values{"tagid": []string{strconv.Itoa(id)}},
		nil,
	)
}

func (c *TagClient) List(ctx context.Context) ([]TagInfo, error) {
	var result struct {
		Tags []TagInfo `json:"taglist"`
	}
	err := c.api.Get(ctx, "work.contact.tag.list", "cgi-bin/tag/list", nil, &result)
	return result.Tags, err
}

// BatchClient manages asynchronous contact import and export jobs.
type BatchClient struct {
	api            *api.Client
	encodingAESKey string
}

func (c *BatchClient) SyncUsers(
	ctx context.Context,
	mediaID string,
	toInvite bool,
	callback *BatchCallback,
) (string, error) {
	return c.importJob(ctx, "syncuser", mediaID, toInvite, callback)
}

func (c *BatchClient) ReplaceUsers(
	ctx context.Context,
	mediaID string,
	toInvite bool,
	callback *BatchCallback,
) (string, error) {
	return c.importJob(ctx, "replaceuser", mediaID, toInvite, callback)
}

func (c *BatchClient) ReplaceDepartments(
	ctx context.Context,
	mediaID string,
	callback *BatchCallback,
) (string, error) {
	return c.importJob(ctx, "replaceparty", mediaID, false, callback)
}

func (c *BatchClient) importJob(
	ctx context.Context,
	action string,
	mediaID string,
	toInvite bool,
	callback *BatchCallback,
) (string, error) {
	body := struct {
		MediaID  string         `json:"media_id"`
		ToInvite bool           `json:"to_invite,omitempty"`
		Callback *BatchCallback `json:"callback,omitempty"`
	}{
		MediaID:  mediaID,
		ToInvite: toInvite,
		Callback: callback,
	}
	var result struct {
		JobID string `json:"jobid"`
	}
	err := c.api.Post(
		ctx,
		"work.contact.batch."+action,
		"cgi-bin/batch/"+action,
		body,
		&result,
	)
	return result.JobID, err
}

func (c *BatchClient) Result(ctx context.Context, jobID string) (*BatchResult, error) {
	result := new(BatchResult)
	err := c.api.Get(
		ctx,
		"work.contact.batch.result",
		"cgi-bin/batch/getresult",
		url.Values{"jobid": []string{jobID}},
		result,
	)
	return result, err
}

func (c *BatchClient) ExportUsers(ctx context.Context, blockSize int) (string, error) {
	return c.exportJob(ctx, "user", blockSize, 0)
}

func (c *BatchClient) ExportSimpleUsers(ctx context.Context, blockSize int) (string, error) {
	return c.exportJob(ctx, "simple_user", blockSize, 0)
}

func (c *BatchClient) ExportDepartments(ctx context.Context, blockSize int) (string, error) {
	return c.exportJob(ctx, "department", blockSize, 0)
}

func (c *BatchClient) ExportTagUsers(
	ctx context.Context,
	blockSize int,
	tagID int,
) (string, error) {
	return c.exportJob(ctx, "taguser", blockSize, tagID)
}

func (c *BatchClient) exportJob(
	ctx context.Context,
	action string,
	blockSize int,
	tagID int,
) (string, error) {
	body := struct {
		EncodingAESKey string `json:"encoding_aeskey"`
		BlockSize      int    `json:"block_size"`
		TagID          int    `json:"tagid,omitempty"`
	}{
		EncodingAESKey: base64.StdEncoding.EncodeToString([]byte(c.encodingAESKey)),
		BlockSize:      blockSize,
		TagID:          tagID,
	}
	var result struct {
		JobID string `json:"jobid"`
	}
	err := c.api.Post(
		ctx,
		"work.contact.export."+action,
		"cgi-bin/export/"+action,
		body,
		&result,
	)
	return result.JobID, err
}

func (c *BatchClient) ExportResult(ctx context.Context, jobID string) (*ExportResult, error) {
	result := new(ExportResult)
	err := c.api.Get(
		ctx,
		"work.contact.export.result",
		"cgi-bin/export/get_result",
		url.Values{"jobid": []string{jobID}},
		result,
	)
	return result, err
}
