package customer

// ExternalContactInfo 外部联系人信息
type ExternalContactInfo struct {
	ExternalUserid  string           `json:"external_userid"`
	Name            string           `json:"name"`
	Position        string           `json:"position"`
	Avatar          string           `json:"avatar"`
	CorpName        string           `json:"corp_name"`
	CorpFullName    string           `json:"corp_full_name"`
	Type            int              `json:"type"`
	Gender          int              `json:"gender"`
	Unionid         string           `json:"unionid"`
	ExternalProfile *ExternalProfile `json:"external_profile,omitempty"`
}

// ExternalProfile 外部联系人扩展属性
type ExternalProfile struct {
	ExternalAttr []ExternalAttr `json:"external_attr,omitempty"`
}

// ExternalAttr 外部联系人扩展属性项
type ExternalAttr struct {
	Type int    `json:"type"`
	Name string `json:"name"`
	Text *struct {
		Value string `json:"value"`
	} `json:"text,omitempty"`
	Web *struct {
		Url   string `json:"url"`
		Title string `json:"title"`
	} `json:"web,omitempty"`
	MiniProgram *struct {
		Appid    string `json:"appid"`
		Pagepath string `json:"pagepath"`
		Title    string `json:"title"`
	} `json:"miniprogram,omitempty"`
}

// FollowUser 跟进成员信息
type FollowUser struct {
	Userid         string          `json:"userid"`
	Remark         string          `json:"remark"`
	Description    string          `json:"description"`
	Createtime     int64           `json:"createtime"`
	Tags           []FollowUserTag `json:"tags,omitempty"`
	RemarkCorpName string          `json:"remark_corp_name"`
	RemarkMobiles  []string        `json:"remark_mobiles"`
	OperUserid     string          `json:"oper_userid"`
	AddWay         int             `json:"add_way"`
	WechatChannels *WechatChannels `json:"wechat_channels,omitempty"`
	State          string          `json:"state"`
}

// FollowUserTag 跟进成员标签
type FollowUserTag struct {
	GroupName string `json:"group_name"`
	TagName   string `json:"tag_name"`
	TagId     string `json:"tag_id"`
	Type      int    `json:"type"`
}

// WechatChannels 视频号信息
type WechatChannels struct {
	Nickname string `json:"nickname"`
	Source   int    `json:"source"`
}

// ExternalContactDetail 客户详情
type ExternalContactDetail struct {
	ExternalContact ExternalContactInfo `json:"external_contact"`
	FollowUser      []FollowUser        `json:"follow_user"`
	NextCursor      string              `json:"next_cursor"`
}

// RemarkRequest 修改客户备注信息请求
type RemarkRequest struct {
	Userid           string   `json:"userid"`
	ExternalUserid   string   `json:"external_userid"`
	Remark           string   `json:"remark,omitempty"`
	Description      string   `json:"description,omitempty"`
	RemarkCompany    string   `json:"remark_company,omitempty"`
	RemarkMobiles    []string `json:"remark_mobiles,omitempty"`
	RemarkPicMediaid string   `json:"remark_pic_mediaid,omitempty"`
}

// ========== 客户标签相关类型 ==========

// TagGroup 标签组
type TagGroup struct {
	GroupId    string    `json:"group_id"`
	GroupName  string    `json:"group_name"`
	CreateTime int64     `json:"create_time"`
	Order      int       `json:"order"`
	Deleted    bool      `json:"deleted"`
	Tag        []TagItem `json:"tag"`
}

// TagItem 标签项
type TagItem struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	CreateTime int64  `json:"create_time"`
	Order      int    `json:"order"`
	Deleted    bool   `json:"deleted"`
}

// GetCorpTagListRequest 获取企业标签库请求
type GetCorpTagListRequest struct {
	TagId   []string `json:"tag_id,omitempty"`
	GroupId []string `json:"group_id,omitempty"`
}

// AddCorpTagRequest 添加企业客户标签请求
type AddCorpTagRequest struct {
	GroupId   string           `json:"group_id,omitempty"`
	GroupName string           `json:"group_name,omitempty"`
	Order     int              `json:"order,omitempty"`
	Tag       []AddCorpTagItem `json:"tag"`
	Agentid   int64            `json:"agentid,omitempty"`
}

// AddCorpTagItem 添加标签项
type AddCorpTagItem struct {
	Name  string `json:"name"`
	Order int    `json:"order,omitempty"`
}

// EditCorpTagRequest 编辑企业客户标签请求
type EditCorpTagRequest struct {
	Id      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Order   int    `json:"order,omitempty"`
	Agentid int64  `json:"agentid,omitempty"`
}

// DelCorpTagRequest 删除企业客户标签请求
type DelCorpTagRequest struct {
	TagId   []string `json:"tag_id,omitempty"`
	GroupId []string `json:"group_id,omitempty"`
	Agentid int64    `json:"agentid,omitempty"`
}

// MarkTagRequest 编辑客户企业标签请求
type MarkTagRequest struct {
	Userid         string   `json:"userid"`
	ExternalUserid string   `json:"external_userid"`
	AddTag         []string `json:"add_tag,omitempty"`
	RemoveTag      []string `json:"remove_tag,omitempty"`
}

// ========== 批量获取客户详情相关类型 ==========

// BatchGetByUserRequest 批量获取客户详情请求
type BatchGetByUserRequest struct {
	UseridList []string `json:"userid_list"`
	Cursor     string   `json:"cursor,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

// ExternalContactBatchInfo 批量获取的客户详情项
type ExternalContactBatchInfo struct {
	ExternalContact ExternalContactInfo `json:"external_contact"`
	FollowInfo      FollowUser          `json:"follow_info"`
}

// BatchGetByUserResult 批量获取客户详情结果
type BatchGetByUserResult struct {
	ExternalContactList []ExternalContactBatchInfo `json:"external_contact_list"`
	NextCursor          string                     `json:"next_cursor"`
}

// ========== unionid转换相关类型 ==========

// UnionidToExternalUseridRequest unionid转换请求
type UnionidToExternalUseridRequest struct {
	Unionid string `json:"unionid"`
	Openid  string `json:"openid,omitempty"`
}

// ExternalUseridInfo external_userid信息
type ExternalUseridInfo struct {
	CorpName       string `json:"corp_name"`
	ExternalUserid string `json:"external_userid"`
}

// ========== 客户联系规则组相关类型 ==========

// StrategyPrivilege 规则组权限
type StrategyPrivilege struct {
	ViewCustomerList        bool `json:"view_customer_list"`
	ViewCustomerData        bool `json:"view_customer_data"`
	ViewRoomList            bool `json:"view_room_list"`
	ContactMe               bool `json:"contact_me"`
	JoinRoom                bool `json:"join_room"`
	ShareCustomer           bool `json:"share_customer"`
	OperResignCustomer      bool `json:"oper_resign_customer"`
	OperResignGroup         bool `json:"oper_resign_group"`
	SendCustomerMsg         bool `json:"send_customer_msg"`
	EditWelcomeMsg          bool `json:"edit_welcome_msg"`
	ViewBehaviorData        bool `json:"view_behavior_data"`
	ViewRoomData            bool `json:"view_room_data"`
	SendGroupMsg            bool `json:"send_group_msg"`
	RoomDeduplication       bool `json:"room_deduplication"`
	RapidReply              bool `json:"rapid_reply"`
	OnjobCustomerTransfer   bool `json:"onjob_customer_transfer"`
	EditAntiSpamRule        bool `json:"edit_anti_spam_rule"`
	ExportCustomerList      bool `json:"export_customer_list"`
	ExportCustomerData      bool `json:"export_customer_data"`
	ExportCustomerGroupList bool `json:"export_customer_group_list"`
	ManageCustomerTag       bool `json:"manage_customer_tag"`
}

// StrategyRangeItem 规则组管理范围节点
type StrategyRangeItem struct {
	Type    int    `json:"type"`
	Userid  string `json:"userid,omitempty"`
	Partyid int    `json:"partyid,omitempty"`
}

// StrategyInfo 规则组详情
type StrategyInfo struct {
	StrategyId   int               `json:"strategy_id"`
	ParentId     int               `json:"parent_id"`
	StrategyName string            `json:"strategy_name"`
	CreateTime   int64             `json:"create_time"`
	AdminList    []string          `json:"admin_list"`
	Privilege    StrategyPrivilege `json:"privilege"`
}

// StrategyListItem 规则组列表项
type StrategyListItem struct {
	StrategyId int `json:"strategy_id"`
}

// StrategyListResult 获取规则组列表结果
type StrategyListResult struct {
	Strategy   []StrategyListItem `json:"strategy"`
	NextCursor string             `json:"next_cursor"`
}

// StrategyGetRangeRequest 获取规则组管理范围请求
type StrategyGetRangeRequest struct {
	StrategyId int    `json:"strategy_id"`
	Cursor     string `json:"cursor,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// StrategyGetRangeResult 获取规则组管理范围结果
type StrategyGetRangeResult struct {
	Range      []StrategyRangeItem `json:"range"`
	NextCursor string              `json:"next_cursor"`
}

// CreateStrategyRequest 创建规则组请求
type CreateStrategyRequest struct {
	ParentId     int                 `json:"parent_id,omitempty"`
	StrategyName string              `json:"strategy_name"`
	AdminList    []string            `json:"admin_list"`
	Privilege    *StrategyPrivilege  `json:"privilege,omitempty"`
	Range        []StrategyRangeItem `json:"range"`
}

// EditStrategyRequest 编辑规则组请求
type EditStrategyRequest struct {
	StrategyId   int                 `json:"strategy_id"`
	StrategyName string              `json:"strategy_name,omitempty"`
	AdminList    []string            `json:"admin_list,omitempty"`
	Privilege    *StrategyPrivilege  `json:"privilege,omitempty"`
	RangeAdd     []StrategyRangeItem `json:"range_add,omitempty"`
	RangeDel     []StrategyRangeItem `json:"range_del,omitempty"`
}

// ========== 客户群管理相关类型 ==========

// OwnerFilter 群主过滤
type OwnerFilter struct {
	UseridList []string `json:"userid_list"`
}

// GroupChatListRequest 获取客户群列表请求
type GroupChatListRequest struct {
	StatusFilter int          `json:"status_filter"`
	OwnerFilter  *OwnerFilter `json:"owner_filter,omitempty"`
	Cursor       string       `json:"cursor,omitempty"`
	Limit        int          `json:"limit"`
}

// GroupChatListItem 客户群列表项
type GroupChatListItem struct {
	ChatId string `json:"chat_id"`
	Status int    `json:"status"`
}

// GroupChatListResult 获取客户群列表结果
type GroupChatListResult struct {
	GroupChatList []GroupChatListItem `json:"group_chat_list"`
	NextCursor    string              `json:"next_cursor"`
}

// GroupChatGetRequest 获取客户群详情请求
type GroupChatGetRequest struct {
	ChatId   string `json:"chat_id"`
	NeedName int    `json:"need_name,omitempty"`
}

// GroupChatMemberInvitor 邀请者
type GroupChatMemberInvitor struct {
	Userid string `json:"userid"`
}

// GroupChatMember 群成员
type GroupChatMember struct {
	Userid        string                  `json:"userid"`
	Type          int                     `json:"type"`
	Unionid       string                  `json:"unionid,omitempty"`
	JoinTime      int64                   `json:"join_time"`
	JoinScene     int                     `json:"join_scene"`
	Invitor       *GroupChatMemberInvitor `json:"invitor,omitempty"`
	GroupNickname string                  `json:"group_nickname"`
	Name          string                  `json:"name,omitempty"`
	State         string                  `json:"state,omitempty"`
}

// GroupChatAdmin 群管理员
type GroupChatAdmin struct {
	Userid string `json:"userid"`
}

// GroupChatJoinWayRequest 配置客户群进群方式请求
type GroupChatJoinWayRequest struct {
	Scene          int      `json:"scene"`
	Remark         string   `json:"remark,omitempty"`
	AutoCreateRoom int      `json:"auto_create_room,omitempty"`
	RoomBaseName   string   `json:"room_base_name,omitempty"`
	RoomBaseId     int      `json:"room_base_id,omitempty"`
	ChatIdList     []string `json:"chat_id_list"`
	State          string   `json:"state,omitempty"`
	MarkSource     *bool    `json:"mark_source,omitempty"`
}

// GroupChatJoinWayUpdateRequest 更新客户群进群方式请求
type GroupChatJoinWayUpdateRequest struct {
	ConfigId       string   `json:"config_id"`
	Scene          int      `json:"scene"`
	Remark         string   `json:"remark,omitempty"`
	AutoCreateRoom int      `json:"auto_create_room,omitempty"`
	RoomBaseName   string   `json:"room_base_name,omitempty"`
	RoomBaseId     int      `json:"room_base_id,omitempty"`
	ChatIdList     []string `json:"chat_id_list"`
	State          string   `json:"state,omitempty"`
	MarkSource     *bool    `json:"mark_source,omitempty"`
}

// GroupChatJoinWay 客户群进群方式详情
type GroupChatJoinWay struct {
	ConfigId       string   `json:"config_id"`
	Scene          int      `json:"scene"`
	Remark         string   `json:"remark"`
	AutoCreateRoom int      `json:"auto_create_room"`
	RoomBaseName   string   `json:"room_base_name"`
	RoomBaseId     int      `json:"room_base_id"`
	ChatIdList     []string `json:"chat_id_list"`
	QrCode         string   `json:"qr_code"`
	State          string   `json:"state"`
	MarkSource     bool     `json:"mark_source"`
}

// GroupChatDetail 客户群详情
type GroupChatDetail struct {
	ChatId        string            `json:"chat_id"`
	Name          string            `json:"name"`
	Owner         string            `json:"owner"`
	CreateTime    int64             `json:"create_time"`
	Notice        string            `json:"notice"`
	MemberList    []GroupChatMember `json:"member_list"`
	AdminList     []GroupChatAdmin  `json:"admin_list"`
	MemberVersion string            `json:"member_version"`
}

// ========== 内部响应类型 ==========

// followUserListResult 获取配置了客户联系功能的成员列表响应
type followUserListResult struct {
	FollowUser []string `json:"follow_user"`
}

// externalUseridListResult 获取客户列表响应
type externalUseridListResult struct {
	ExternalUserid []string `json:"external_userid"`
}

// externalContactDetailResult 获取客户详情响应
type externalContactDetailResult struct {
	ExternalContactDetail
}

// corpTagListResult 获取企业标签库响应
type corpTagListResult struct {
	TagGroup []TagGroup `json:"tag_group"`
}

// addCorpTagResult 添加企业客户标签响应
type addCorpTagResult struct {
	TagGroup TagGroup `json:"tag_group"`
}

// batchGetByUserResult 批量获取客户详情响应
type batchGetByUserResult struct {
	BatchGetByUserResult
}

// unionidToExternalUseridResult unionid转换响应
type unionidToExternalUseridResult struct {
	ExternalUseridInfo []ExternalUseridInfo `json:"external_userid_info"`
}

// toServiceExternalUseridResult 代开发应用external_userid转换响应
type toServiceExternalUseridResult struct {
	ExternalUserid string `json:"external_userid"`
}

// strategyListResponse 获取规则组列表响应
type strategyListResponse struct {
	StrategyListResult
}

// strategyGetResponse 获取规则组详情响应
type strategyGetResponse struct {
	Strategy StrategyInfo `json:"strategy"`
}

// strategyGetRangeResponse 获取规则组管理范围响应
type strategyGetRangeResponse struct {
	StrategyGetRangeResult
}

// strategyCreateResponse 创建规则组响应
type strategyCreateResponse struct {
	StrategyId int `json:"strategy_id"`
}

// groupChatListResponse 获取客户群列表响应
type groupChatListResponse struct {
	GroupChatListResult
}

// groupChatGetResponse 获取客户群详情响应
type groupChatGetResponse struct {
	GroupChat GroupChatDetail `json:"group_chat"`
}

// opengidToChatIdResponse 客户群opengid转换响应
type opengidToChatIdResponse struct {
	ChatId string `json:"chat_id"`
}

// groupChatAddJoinWayResponse 配置客户群进群方式响应
type groupChatAddJoinWayResponse struct {
	ConfigId string `json:"config_id"`
}

// groupChatGetJoinWayResponse 获取客户群进群方式配置响应
type groupChatGetJoinWayResponse struct {
	JoinWay GroupChatJoinWay `json:"join_way"`
}
