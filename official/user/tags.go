package user

import (
	"context"
	"strconv"
)

// Tag describes an official account user tag.
type Tag struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// TaggedUsers is one page of users associated with a tag.
type TaggedUsers struct {
	Count int `json:"count"`
	Data  struct {
		OpenIDs []string `json:"openid"`
	} `json:"data"`
	NextOpenID string `json:"next_openid"`
}

// TagClient manages official account user tags.
type TagClient struct {
	api Caller
}

// Create creates a user tag.
func (c *TagClient) Create(ctx context.Context, name string) (Tag, error) {
	var result struct {
		Tag Tag `json:"tag"`
	}
	err := c.api.Post(
		ctx,
		"official.user.tag.create",
		"cgi-bin/tags/create",
		map[string]interface{}{"tag": map[string]string{"name": name}},
		&result,
	)
	return result.Tag, err
}

// List returns all user tags.
func (c *TagClient) List(ctx context.Context) ([]Tag, error) {
	var result struct {
		Tags []Tag `json:"tags"`
	}
	err := c.api.Get(ctx, "official.user.tag.list", "cgi-bin/tags/get", nil, &result)
	return result.Tags, err
}

// Update changes a tag name.
func (c *TagClient) Update(ctx context.Context, tagID int, name string) error {
	return c.api.Post(
		ctx,
		"official.user.tag.update",
		"cgi-bin/tags/update",
		map[string]interface{}{
			"tag": map[string]interface{}{"id": tagID, "name": name},
		},
		nil,
	)
}

// Delete removes a user tag.
func (c *TagClient) Delete(ctx context.Context, tagID int) error {
	return c.api.Post(
		ctx,
		"official.user.tag.delete",
		"cgi-bin/tags/delete",
		map[string]interface{}{"tag": map[string]int{"id": tagID}},
		nil,
	)
}

// Users returns users carrying tagID.
func (c *TagClient) Users(
	ctx context.Context,
	tagID int,
	nextOpenID string,
) (*TaggedUsers, error) {
	result := new(TaggedUsers)
	err := c.api.Post(
		ctx,
		"official.user.tag.users",
		"cgi-bin/user/tag/get",
		map[string]interface{}{"tagid": tagID, "next_openid": nextOpenID},
		result,
	)
	return result, err
}

// TagUsers adds tagID to the supplied users.
func (c *TagClient) TagUsers(
	ctx context.Context,
	openIDs []string,
	tagID int,
) error {
	return c.members(ctx, "batchtagging", openIDs, tagID)
}

// UntagUsers removes tagID from the supplied users.
func (c *TagClient) UntagUsers(
	ctx context.Context,
	openIDs []string,
	tagID int,
) error {
	return c.members(ctx, "batchuntagging", openIDs, tagID)
}

// UserTags returns the tag IDs attached to openID.
func (c *TagClient) UserTags(ctx context.Context, openID string) ([]int, error) {
	var result struct {
		TagIDs []int `json:"tagid_list"`
	}
	err := c.api.Post(
		ctx,
		"official.user.tag.user_tags",
		"cgi-bin/tags/getidlist",
		map[string]string{"openid": openID},
		&result,
	)
	return result.TagIDs, err
}

func (c *TagClient) members(
	ctx context.Context,
	action string,
	openIDs []string,
	tagID int,
) error {
	return c.api.Post(
		ctx,
		"official.user.tag."+action+"."+strconv.Itoa(tagID),
		"cgi-bin/tags/members/"+action,
		map[string]interface{}{"openid_list": openIDs, "tagid": tagID},
		nil,
	)
}
