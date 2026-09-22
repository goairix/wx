package kf

import kernelError "github.com/goairix/wx/v2/kernel/error"

// UpdateAccountRequest 修改客服账号请求
type UpdateAccountRequest struct {
	OpenKfid string `json:"open_kfid"`
	Name     string `json:"name,omitempty"`
	MediaId  string `json:"media_id,omitempty"`
}

// AccountInfo 客服账号信息
type AccountInfo struct {
	OpenKfid        string `json:"open_kfid"`
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	ManagePrivilege bool   `json:"manage_privilege"`
}

// AccountListResult 获取客服账号列表结果
type AccountListResult struct {
	AccountList []AccountInfo `json:"account_list"`
}

// ========== 接待人员相关类型 ==========

// ServicerResult 添加/删除接待人员结果项
type ServicerResult struct {
	Userid  string `json:"userid"`
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

// ServicerInfo 接待人员信息
type ServicerInfo struct {
	Userid       string `json:"userid"`
	Status       int    `json:"status"`
	StopType     int    `json:"stop_type,omitempty"`
	DepartmentId int    `json:"department_id,omitempty"`
}

// ========== 会话管理相关类型 ==========

// ServiceStateInfo 会话状态信息
type ServiceStateInfo struct {
	ServiceState   int    `json:"service_state"`
	ServicerUserid string `json:"servicer_userid"`
}

// ServiceStateTransRequest 变更会话状态请求
type ServiceStateTransRequest struct {
	OpenKfid       string `json:"open_kfid"`
	ExternalUserid string `json:"external_userid"`
	ServiceState   int    `json:"service_state"`
	ServicerUserid string `json:"servicer_userid,omitempty"`
}

// ========== 消息收发相关类型 ==========

// SyncMsgRequest 读取消息请求
type SyncMsgRequest struct {
	Cursor      string `json:"cursor,omitempty"`
	Token       string `json:"token,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	VoiceFormat int    `json:"voice_format,omitempty"`
	OpenKfid    string `json:"open_kfid,omitempty"`
}

// SyncMsgItem 消息项
type SyncMsgItem struct {
	MsgId          string           `json:"msgid"`
	OpenKfid       string           `json:"open_kfid"`
	ExternalUserid string           `json:"external_userid"`
	SendTime       int64            `json:"send_time"`
	Origin         int              `json:"origin"`
	ServicerUserid string           `json:"servicer_userid"`
	MsgType        string           `json:"msgtype"`
	Text           *TextContent     `json:"text,omitempty"`
	Image          *MediaContent    `json:"image,omitempty"`
	Voice          *MediaContent    `json:"voice,omitempty"`
	Video          *MediaContent    `json:"video,omitempty"`
	Event          *MsgEventContent `json:"event,omitempty"`
}

// SyncMsgResult 读取消息结果
type SyncMsgResult struct {
	NextCursor string        `json:"next_cursor"`
	HasMore    int           `json:"has_more"`
	MsgList    []SyncMsgItem `json:"msg_list"`
}

// TextContent 文本消息内容
type TextContent struct {
	Content string `json:"content"`
}

// MediaContent 媒体消息内容（图片/语音/视频）
type MediaContent struct {
	MediaId string `json:"media_id"`
}

// MsgEventContent 事件消息内容
type MsgEventContent struct {
	EventType      string `json:"event_type"`
	OpenKfid       string `json:"open_kfid"`
	ExternalUserid string `json:"external_userid"`
	Scene          string `json:"scene,omitempty"`
	SceneParam     string `json:"scene_param,omitempty"`
	WelcomeCode    string `json:"welcome_code,omitempty"`
	WechatChannels *struct {
		Nickname string `json:"nickname,omitempty"`
	} `json:"wechat_channels,omitempty"`
}

// ========== 内部响应类型 ==========

// addAccountResponse 添加客服账号响应
type addAccountResponse struct {
	kernelError.ApiError
	OpenKfid string `json:"open_kfid"`
}

// accountListResponse 获取客服账号列表响应
type accountListResponse struct {
	kernelError.ApiError
	AccountListResult
}

// addContactWayResponse 获取客服账号链接响应
type addContactWayResponse struct {
	kernelError.ApiError
	URL string `json:"url"`
}

// servicerAddDelResponse 添加/删除接待人员响应
type servicerAddDelResponse struct {
	kernelError.ApiError
	ResultList []ServicerResult `json:"result_list"`
}

// servicerListResponse 获取接待人员列表响应
type servicerListResponse struct {
	kernelError.ApiError
	ServicerList []ServicerInfo `json:"servicer_list"`
}

// serviceStateGetResponse 获取会话状态响应
type serviceStateGetResponse struct {
	kernelError.ApiError
	ServiceStateInfo
}

// serviceStateTransResponse 变更会话状态响应
type serviceStateTransResponse struct {
	kernelError.ApiError
	MsgCode string `json:"msg_code"`
}

// sendMsgRequest 发送消息请求
type sendMsgRequest struct {
	ToUser   string        `json:"touser"`
	OpenKfid string        `json:"open_kfid"`
	MsgId    string        `json:"msgid,omitempty"`
	MsgType  string        `json:"msgtype"`
	Text     *TextContent  `json:"text,omitempty"`
	Image    *MediaContent `json:"image,omitempty"`
	Voice    *MediaContent `json:"voice,omitempty"`
	Video    *MediaContent `json:"video,omitempty"`
}

// sendMsgResponse 发送消息响应
type sendMsgResponse struct {
	kernelError.ApiError
	MsgId string `json:"msgid"`
}

// syncMsgResponse 读取消息响应
type syncMsgResponse struct {
	kernelError.ApiError
	SyncMsgResult
}
