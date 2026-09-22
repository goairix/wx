package customer

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/work/http"
)

// GroupChat 客户群管理
type GroupChat struct {
	account contracts.AccountInterface
}

// NewGroupChat 创建 GroupChat 实例。
func NewGroupChat(account contracts.AccountInterface) *GroupChat {
	return &GroupChat{account: account}
}

// List 获取客户群列表
func (g *GroupChat) List(req GroupChatListRequest) (*GroupChatListResult, error) {
	accessToken, err := g.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/groupchat/list?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result groupChatListResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.GroupChatListResult, nil
}

// Get 获取客户群详情
func (g *GroupChat) Get(req GroupChatGetRequest) (*GroupChatDetail, error) {
	accessToken, err := g.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/groupchat/get?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result groupChatGetResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.GroupChat, nil
}

// OpengidToChatId 客户群opengid转换
func (g *GroupChat) OpengidToChatId(opengid string) (string, error) {
	accessToken, err := g.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/opengid_to_chatid?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]string{"opengid": opengid})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result opengidToChatIdResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.ChatId, nil
}

// AddJoinWay 配置客户群进群方式
func (g *GroupChat) AddJoinWay(req GroupChatJoinWayRequest) (string, error) {
	accessToken, err := g.account.AccessToken()
	if err != nil {
		return "", kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/groupchat/add_join_way?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return "", kernelError.New(0, err)
	}

	var result groupChatAddJoinWayResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.ConfigId, nil
}

// GetJoinWay 获取客户群进群方式配置
func (g *GroupChat) GetJoinWay(configId string) (*GroupChatJoinWay, error) {
	accessToken, err := g.account.AccessToken()
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/groupchat/get_join_way?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]string{"config_id": configId})
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	var result groupChatGetJoinWayResponse
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return &result.JoinWay, nil
}

// UpdateJoinWay 更新客户群进群方式配置
func (g *GroupChat) UpdateJoinWay(req GroupChatJoinWayUpdateRequest) error {
	accessToken, err := g.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/groupchat/update_join_way?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, req)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}

// DelJoinWay 删除客户群进群方式配置
func (g *GroupChat) DelJoinWay(configId string) error {
	accessToken, err := g.account.AccessToken()
	if err != nil {
		return kernelError.New(0, err)
	}

	apiUrl := fmt.Sprintf("cgi-bin/externalcontact/groupchat/del_join_way?access_token=%s", accessToken.AccessToken)
	res, err := http.PostJSON(apiUrl, map[string]string{"config_id": configId})
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return kernelError.NewWithApiError(result)
	}

	return nil
}
