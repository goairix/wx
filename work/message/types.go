package message

// Messenger 消息接口
type Messenger interface {
	MsgType() string
}

// SendOption 发送应用消息选项
type SendOption struct {
	ToUser                 string `json:"touser,omitempty"`
	ToParty                string `json:"toparty,omitempty"`
	ToTag                  string `json:"totag,omitempty"`
	AgentId                int64  `json:"agentid"`
	Safe                   int    `json:"safe,omitempty"`
	EnableIdTrans          int    `json:"enable_id_trans,omitempty"`
	EnableDuplicateCheck   int    `json:"enable_duplicate_check,omitempty"`
	DuplicateCheckInterval int    `json:"duplicate_check_interval,omitempty"`
}

// Text 文本消息
type Text struct {
	Content             string   `json:"content"`
	MentionedList       []string `json:"mentioned_list,omitempty"`
	MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
}

// MsgType 返回企业微信消息类型标识。
func (t *Text) MsgType() string { return "text" }

// Image 图片消息
type Image struct {
	MediaId string `json:"media_id"`
}

// MsgType 返回企业微信消息类型标识。
func (i *Image) MsgType() string { return "image" }

// Voice 语音消息
type Voice struct {
	MediaId string `json:"media_id"`
}

// MsgType 返回企业微信消息类型标识。
func (v *Voice) MsgType() string { return "voice" }

// Video 视频消息
type Video struct {
	MediaId     string `json:"media_id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// MsgType 返回企业微信消息类型标识。
func (v *Video) MsgType() string { return "video" }

// File 文件消息
type File struct {
	MediaId string `json:"media_id"`
}

// MsgType 返回企业微信消息类型标识。
func (f *File) MsgType() string { return "file" }

// TextCard 文本卡片消息
type TextCard struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	BtnTxt      string `json:"btntxt,omitempty"`
}

// MsgType 返回企业微信消息类型标识。
func (tc *TextCard) MsgType() string { return "textcard" }

// NewsArticle 图文消息-文章
type NewsArticle struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Url         string `json:"url,omitempty"`
	PicUrl      string `json:"picurl,omitempty"`
	AppId       string `json:"appid,omitempty"`
	PagePath    string `json:"pagepath,omitempty"`
}

// News 图文消息
type News struct {
	Articles []NewsArticle `json:"articles"`
}

// MsgType 返回企业微信消息类型标识。
func (n *News) MsgType() string { return "news" }

// MpNewsArticle mpnews图文消息-文章
type MpNewsArticle struct {
	Title            string `json:"title"`
	ThumbMediaId     string `json:"thumb_media_id"`
	Author           string `json:"author,omitempty"`
	ContentSourceUrl string `json:"content_source_url,omitempty"`
	Content          string `json:"content"`
	Digest           string `json:"digest,omitempty"`
}

// MpNews mpnews图文消息
type MpNews struct {
	Articles []MpNewsArticle `json:"articles"`
}

// MsgType 返回企业微信消息类型标识。
func (m *MpNews) MsgType() string { return "mpnews" }

// Markdown markdown消息
type Markdown struct {
	Content string `json:"content"`
}

// MsgType 返回企业微信消息类型标识。
func (md *Markdown) MsgType() string { return "markdown" }

// MiniProgramNotice 小程序通知消息
type MiniProgramNotice struct {
	AppId             string                  `json:"appid"`
	Page              string                  `json:"page,omitempty"`
	Title             string                  `json:"title"`
	Description       string                  `json:"description,omitempty"`
	EmphasisFirstItem bool                    `json:"emphasis_first_item,omitempty"`
	ContentItem       []MiniProgramNoticeItem `json:"content_item,omitempty"`
}

// MsgType 返回企业微信消息类型标识。
func (mp *MiniProgramNotice) MsgType() string { return "miniprogram_notice" }

// MiniProgramNoticeItem 小程序通知消息-内容项
type MiniProgramNoticeItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// TemplateCard 模版卡片消息
type TemplateCard struct {
	CardType              string                          `json:"card_type"`
	Source                *TemplateCardSource             `json:"source,omitempty"`
	ActionMenu            *TemplateCardActionMenu         `json:"action_menu,omitempty"`
	TaskId                string                          `json:"task_id,omitempty"`
	MainTitle             *TemplateCardTitle              `json:"main_title,omitempty"`
	QuoteArea             *TemplateCardQuoteArea          `json:"quote_area,omitempty"`
	EmphasisContent       *TemplateCardEmphasis           `json:"emphasis_content,omitempty"`
	SubTitleText          string                          `json:"sub_title_text,omitempty"`
	HorizontalContentList []TemplateCardHorizontalContent `json:"horizontal_content_list,omitempty"`
	JumpList              []TemplateCardJump              `json:"jump_list,omitempty"`
	CardAction            *TemplateCardAction             `json:"card_action,omitempty"`
	ButtonSelection       *TemplateCardButtonSelection    `json:"button_selection,omitempty"`
	ButtonList            []TemplateCardButton            `json:"button_list,omitempty"`
	CheckBox              *TemplateCardCheckBox           `json:"checkbox,omitempty"`
	SelectList            []TemplateCardSelect            `json:"select_list,omitempty"`
	SubmitButton          *TemplateCardSubmitButton       `json:"submit_button,omitempty"`
	ImageTextArea         *TemplateCardImageTextArea      `json:"image_text_area,omitempty"`
	CardImage             *TemplateCardCardImage          `json:"card_image,omitempty"`
	VerticalContentList   []TemplateCardVerticalContent   `json:"vertical_content_list,omitempty"`
}

// MsgType 返回企业微信消息类型标识。
func (tc *TemplateCard) MsgType() string { return "template_card" }

// TemplateCardSource 模版卡片-来源
type TemplateCardSource struct {
	IconUrl   string `json:"icon_url,omitempty"`
	Desc      string `json:"desc,omitempty"`
	DescColor int    `json:"desc_color,omitempty"`
}

// TemplateCardActionMenu 模版卡片-操作菜单
type TemplateCardActionMenu struct {
	Desc       string                       `json:"desc,omitempty"`
	ActionList []TemplateCardActionMenuItem `json:"action_list"`
}

// TemplateCardActionMenuItem 模版卡片-操作菜单项
type TemplateCardActionMenuItem struct {
	Text string `json:"text"`
	Key  string `json:"key"`
}

// TemplateCardTitle 模版卡片-主标题
type TemplateCardTitle struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

// TemplateCardQuoteArea 模版卡片-引用区域
type TemplateCardQuoteArea struct {
	Type      int    `json:"type,omitempty"`
	Url       string `json:"url,omitempty"`
	AppId     string `json:"appid,omitempty"`
	PagePath  string `json:"pagepath,omitempty"`
	Title     string `json:"title,omitempty"`
	QuoteText string `json:"quote_text,omitempty"`
}

// TemplateCardEmphasis 模版卡片-关键数据
type TemplateCardEmphasis struct {
	Title string `json:"title,omitempty"`
	Desc  string `json:"desc,omitempty"`
}

// TemplateCardHorizontalContent 模版卡片-二级标题+文本
type TemplateCardHorizontalContent struct {
	KeyName string `json:"keyname,omitempty"`
	Value   string `json:"value,omitempty"`
	Type    int    `json:"type,omitempty"`
	Url     string `json:"url,omitempty"`
	MediaId string `json:"media_id,omitempty"`
	Userid  string `json:"userid,omitempty"`
}

// TemplateCardJump 模版卡片-跳转
type TemplateCardJump struct {
	Type     int    `json:"type,omitempty"`
	Url      string `json:"url,omitempty"`
	Title    string `json:"title"`
	AppId    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

// TemplateCardAction 模版卡片-整体卡片点击
type TemplateCardAction struct {
	Type     int    `json:"type,omitempty"`
	Url      string `json:"url,omitempty"`
	AppId    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

// TemplateCardButtonSelection 模版卡片-下拉式按钮
type TemplateCardButtonSelection struct {
	QuestionKey string                              `json:"question_key"`
	Title       string                              `json:"title,omitempty"`
	OptionList  []TemplateCardButtonSelectionOption `json:"option_list"`
	SelectedId  string                              `json:"selected_id,omitempty"`
}

// TemplateCardButtonSelectionOption 模版卡片-下拉式按钮选项
type TemplateCardButtonSelectionOption struct {
	Id   string `json:"id"`
	Text string `json:"text"`
}

// TemplateCardButton 模版卡片-按钮
type TemplateCardButton struct {
	Text  string `json:"text"`
	Style int    `json:"style,omitempty"`
	Key   string `json:"key,omitempty"`
	Type  int    `json:"type,omitempty"`
	Url   string `json:"url,omitempty"`
}

// TemplateCardCheckBox 模版卡片-选择题
type TemplateCardCheckBox struct {
	QuestionKey string                       `json:"question_key"`
	OptionList  []TemplateCardCheckBoxOption `json:"option_list"`
	Mode        int                          `json:"mode,omitempty"`
}

// TemplateCardCheckBoxOption 模版卡片-选择题选项
type TemplateCardCheckBoxOption struct {
	Id        string `json:"id"`
	Text      string `json:"text"`
	IsChecked bool   `json:"is_checked,omitempty"`
}

// TemplateCardSelect 模版卡片-下拉选择器
type TemplateCardSelect struct {
	QuestionKey string                     `json:"question_key"`
	Title       string                     `json:"title,omitempty"`
	SelectedId  string                     `json:"selected_id,omitempty"`
	OptionList  []TemplateCardSelectOption `json:"option_list"`
}

// TemplateCardSelectOption 模版卡片-下拉选择器选项
type TemplateCardSelectOption struct {
	Id   string `json:"id"`
	Text string `json:"text"`
}

// TemplateCardSubmitButton 模版卡片-提交按钮
type TemplateCardSubmitButton struct {
	Text string `json:"text"`
	Key  string `json:"key"`
}

// TemplateCardImageTextArea 模版卡片-左图右文
type TemplateCardImageTextArea struct {
	Type     int    `json:"type,omitempty"`
	Url      string `json:"url,omitempty"`
	AppId    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
	Title    string `json:"title,omitempty"`
	Desc     string `json:"desc,omitempty"`
	ImageUrl string `json:"image_url"`
}

// TemplateCardCardImage 模版卡片-图片
type TemplateCardCardImage struct {
	Url         string  `json:"url"`
	AspectRatio float64 `json:"aspect_ratio,omitempty"`
}

// TemplateCardVerticalContent 模版卡片-纵向内容
type TemplateCardVerticalContent struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

// SendResult 发送应用消息结果
type SendResult struct {
	InvalidUser    string `json:"invaliduser"`
	InvalidParty   string `json:"invalidparty"`
	InvalidTag     string `json:"invalidtag"`
	UnlicensedUser string `json:"unlicenseduser"`
	MsgId          string `json:"msgid"`
	ResponseCode   string `json:"response_code"`
}

// sendResponse 发送应用消息响应
type sendResponse struct {
	SendResult
}

// UpdateTemplateCardRequest 更新模版卡片消息请求
type UpdateTemplateCardRequest struct {
	UserIds      []string      `json:"userids,omitempty"`
	PartyIds     []int         `json:"partyids,omitempty"`
	TagIds       []int         `json:"tagids,omitempty"`
	AtAll        int           `json:"atall,omitempty"`
	AgentId      int64         `json:"agentid"`
	ResponseCode string        `json:"response_code"`
	Button       *UpdateButton `json:"button,omitempty"`
	TemplateCard *TemplateCard `json:"template_card,omitempty"`
}

// UpdateButton 更新模版卡片-按钮
type UpdateButton struct {
	ReplaceName string `json:"replace_name"`
}

// UpdateTemplateCardResult 更新模版卡片消息结果
type UpdateTemplateCardResult struct {
	InvalidUser []string `json:"invaliduser"`
}

// updateTemplateCardResponse 更新模版卡片消息响应
type updateTemplateCardResponse struct {
	UpdateTemplateCardResult
}

// RecallRequest 撤回应用消息请求
type RecallRequest struct {
	MsgId string `json:"msgid"`
}

// recallResponse 撤回应用消息响应
type recallResponse struct {
}

// ========== 群聊会话相关类型 ==========

// CreateChatRequest 创建群聊会话请求
type CreateChatRequest struct {
	Name     string   `json:"name,omitempty"`
	Owner    string   `json:"owner,omitempty"`
	UserList []string `json:"userlist"`
	ChatId   string   `json:"chatid,omitempty"`
}

// createChatResponse 创建群聊会话响应
type createChatResponse struct {
	ChatId string `json:"chatid"`
}

// UpdateChatRequest 修改群聊会话请求
type UpdateChatRequest struct {
	ChatId      string   `json:"chatid"`
	Name        string   `json:"name,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	AddUserList []string `json:"add_user_list,omitempty"`
	DelUserList []string `json:"del_user_list,omitempty"`
}

// ChatInfo 群聊会话信息
type ChatInfo struct {
	ChatId   string   `json:"chatid"`
	Name     string   `json:"name"`
	Owner    string   `json:"owner"`
	UserList []string `json:"userlist"`
}

// getChatResponse 获取群聊会话响应
type getChatResponse struct {
	ChatInfo ChatInfo `json:"chat_info"`
}
