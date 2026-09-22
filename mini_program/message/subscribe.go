package message

import (
	"encoding/json"
	"fmt"

	"github.com/goairix/wx/v2/kernel/contracts"
	kernelError "github.com/goairix/wx/v2/kernel/error"
	"github.com/goairix/wx/v2/support/http"
	"github.com/pkg/errors"
)

// Subscribe 小程序订阅消息
type Subscribe struct {
	account contracts.AccountInterface
}

// Message 消息体
type Message struct {
	ToUser           string                `json:"touser"`
	TemplateId       string                `json:"template_id"`
	Page             string                `json:"page"`
	Data             map[string]*DataValue `json:"data,omitempty"`
	MiniProgramState string                `json:"miniprogram_state"`
	Lang             string                `json:"lang"`
}

type DataValue struct {
	Value string `json:"value"`
}

// Category 模板所属类目
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Keyword 模板关键词
type Keyword struct {
	Tid     string `json:"tid"`
	Name    string `json:"name"`
	Example string `json:"example"`
	Rule    string `json:"rule"`
}

// PublicTemplate 公共模板
type PublicTemplate struct {
	Tid        string `json:"tid"`
	Title      string `json:"title"`
	Type       int    `json:"type"`
	CategoryId string `json:"categoryId"`
}

// PrivateTemplate 个人模板
type PrivateTemplate struct {
	PrivateTemplateID    string             `json:"priTmplId"`
	Title                string             `json:"title"`
	Content              string             `json:"content"`
	Example              string             `json:"example"`
	Type                 int                `json:"type"`
	KeywordEnumValueList []KeywordEnumValue `json:"keywordEnumValueList"`
}

type KeywordEnumValue struct {
	EnumValueList []string `json:"enumValueList"`
	KeywordCode   string   `json:"keywordCode"`
}

func NewSubscribe(account contracts.AccountInterface) *Subscribe {
	return &Subscribe{account: account}
}

// GetCategory 获取类目
func (m *Subscribe) GetCategory() ([]Category, error) {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("wxaapi/newtmpl/getcategory?access_token=%s", accessToken.AccessToken)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type categoryResult struct {
		kernelError.ApiError
		Data []Category `json:"data"`
	}
	var result categoryResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Data, nil
}

// GetKeywords 获取关键词列表
func (m *Subscribe) GetKeywords(tid string) ([]Keyword, error) {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("wxaapi/newtmpl/getpubtemplatekeywords?access_token=%s&tid=%s", accessToken.AccessToken, tid)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type categoryResult struct {
		kernelError.ApiError
		Data []Keyword `json:"data"`
	}
	var result categoryResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Data, nil
}

// GetPublicTemplates 获取所属类目下的公共模板
func (m *Subscribe) GetPublicTemplates(ids string, start, limit int) ([]PublicTemplate, error) {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("wxaapi/newtmpl/getpubtemplatetitles?access_token=%s&ids=%s&start=%d&limit=%d", accessToken.AccessToken, ids, start, limit)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type templateResult struct {
		kernelError.ApiError
		Data []PublicTemplate `json:"data"`
	}
	var result templateResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Data, nil
}

// GetTemplateList 获取个人模板列表
func (m *Subscribe) GetTemplateList() ([]PrivateTemplate, error) {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("wxaapi/newtmpl/gettemplate?access_token=%s", accessToken.AccessToken)

	res, err := http.Get(apiUrl)
	if err != nil {
		return nil, kernelError.New(0, err)
	}

	// 返回信息
	type templateResult struct {
		kernelError.ApiError
		Data []PrivateTemplate `json:"data"`
	}
	var result templateResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return nil, kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return nil, kernelError.NewWithApiError(result.ApiError)
	}

	return result.Data, nil
}

// AddTemplate 添加订阅消息模板
func (m *Subscribe) AddTemplate(tid string, kidList []int, sceneDesc string) (string, error) {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("wxaapi/newtmpl/addtemplate?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"tid": tid, "kidList": kidList, "sceneDesc": sceneDesc})
	if err != nil {
		return "", kernelError.New(0, err)
	}

	// 返回信息
	type tempResult struct {
		kernelError.ApiError
		PrivateTmplId string `json:"priTmplId"`
	}

	var result tempResult
	err = json.Unmarshal(res, &result)
	if err != nil {
		return "", kernelError.New(0, err)
	}
	if result.ErrCode != 0 {
		return "", kernelError.NewWithApiError(result.ApiError)
	}

	return result.PrivateTmplId, nil
}

// DeleteTemplate 删除订阅消息模板
func (m *Subscribe) DeleteTemplate(privateTemplateId string) error {
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf("wxaapi/newtmpl/deltemplate?access_token=%s", accessToken.AccessToken)

	res, err := http.PostJSON(apiUrl, map[string]interface{}{"priTmplId": privateTemplateId})
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

// Send 发送订阅消息
func (m *Subscribe) Send(message Message) error {
	if message.ToUser == "" {
		return kernelError.New(0, errors.New("attribute touser can not be empty!"))
	}
	if message.TemplateId == "" {
		return kernelError.New(0, errors.New("attribute template_id can not be empty!"))
	}
	accessToken, _ := m.account.AccessToken()
	apiUrl := fmt.Sprintf(
		"cgi-bin/message/subscribe/send?access_token=%s",
		accessToken.AccessToken,
	)

	if message.MiniProgramState == "" {
		message.MiniProgramState = "formal"
	}
	if message.Lang == "" {
		message.Lang = "zh_CN"
	}

	res, err := http.PostJSON(apiUrl, message)
	if err != nil {
		return kernelError.New(0, err)
	}

	var result kernelError.ApiError
	err = json.Unmarshal(res, &result)
	if err != nil {
		return kernelError.New(0, err)
	}

	return kernelError.NewWithApiError(result)
}
