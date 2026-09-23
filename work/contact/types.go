package contact

// Attr 扩展属性
type Attr struct {
	Type int    `json:"type"`
	Name string `json:"name"`
	Text *struct {
		Value string `json:"value"`
	} `json:"text,omitempty"`
	Web *struct {
		Url   string `json:"url"`
		Title string `json:"title"`
	} `json:"web,omitempty"`
}

// ExternalProfile 对外职务信息
type ExternalProfile struct {
	ExternalCorpName string `json:"external_corp_name,omitempty"`
	WechatChannels   *struct {
		Nickname string `json:"nickname"`
	} `json:"wechat_channels,omitempty"`
	ExternalAttr []Attr `json:"external_attr,omitempty"`
}

// CreateUserRequest 创建成员请求
type CreateUserRequest struct {
	Userid           string           `json:"userid"`
	Name             string           `json:"name"`
	Alias            string           `json:"alias,omitempty"`
	Mobile           string           `json:"mobile,omitempty"`
	Department       []int            `json:"department"`
	Order            []int            `json:"order,omitempty"`
	Position         string           `json:"position,omitempty"`
	Gender           string           `json:"gender,omitempty"`
	Email            string           `json:"email,omitempty"`
	BizMail          string           `json:"biz_mail,omitempty"`
	IsLeaderInDept   []int            `json:"is_leader_in_dept,omitempty"`
	DirectLeader     []string         `json:"direct_leader,omitempty"`
	Enable           int              `json:"enable,omitempty"`
	AvatarMediaid    string           `json:"avatar_mediaid,omitempty"`
	Telephone        string           `json:"telephone,omitempty"`
	Address          string           `json:"address,omitempty"`
	MainDepartment   int              `json:"main_department,omitempty"`
	Extattr          *ExtAttr         `json:"extattr,omitempty"`
	ToInvite         *bool            `json:"to_invite,omitempty"`
	ExternalPosition string           `json:"external_position,omitempty"`
	ExternalProfile  *ExternalProfile `json:"external_profile,omitempty"`
}

// ExtAttr 扩展属性集合
type ExtAttr struct {
	Attrs []Attr `json:"attrs"`
}

// UpdateUserRequest 更新成员请求
type UpdateUserRequest struct {
	Userid           string           `json:"userid"`
	Name             string           `json:"name,omitempty"`
	Alias            string           `json:"alias,omitempty"`
	Mobile           string           `json:"mobile,omitempty"`
	Department       []int            `json:"department,omitempty"`
	Order            []int            `json:"order,omitempty"`
	Position         string           `json:"position,omitempty"`
	Gender           string           `json:"gender,omitempty"`
	Email            string           `json:"email,omitempty"`
	BizMail          string           `json:"biz_mail,omitempty"`
	IsLeaderInDept   []int            `json:"is_leader_in_dept,omitempty"`
	DirectLeader     []string         `json:"direct_leader,omitempty"`
	Enable           int              `json:"enable,omitempty"`
	AvatarMediaid    string           `json:"avatar_mediaid,omitempty"`
	Telephone        string           `json:"telephone,omitempty"`
	Address          string           `json:"address,omitempty"`
	MainDepartment   int              `json:"main_department,omitempty"`
	Extattr          *ExtAttr         `json:"extattr,omitempty"`
	ExternalPosition string           `json:"external_position,omitempty"`
	ExternalProfile  *ExternalProfile `json:"external_profile,omitempty"`
}

// UserInfo 成员详细信息
type UserInfo struct {
	Userid           string           `json:"userid"`
	Name             string           `json:"name"`
	Alias            string           `json:"alias"`
	Mobile           string           `json:"mobile"`
	Department       []int            `json:"department"`
	Order            []int            `json:"order"`
	Position         string           `json:"position"`
	Gender           string           `json:"gender"`
	Email            string           `json:"email"`
	BizMail          string           `json:"biz_mail"`
	IsLeaderInDept   []int            `json:"is_leader_in_dept"`
	DirectLeader     []string         `json:"direct_leader"`
	Avatar           string           `json:"avatar"`
	ThumbAvatar      string           `json:"thumb_avatar"`
	Telephone        string           `json:"telephone"`
	Address          string           `json:"address"`
	OpenUserid       string           `json:"open_userid"`
	MainDepartment   int              `json:"main_department"`
	Extattr          ExtAttr          `json:"extattr"`
	Status           int              `json:"status"`
	QrCode           string           `json:"qr_code"`
	ExternalPosition string           `json:"external_position"`
	ExternalProfile  *ExternalProfile `json:"external_profile"`
}

// UserIdItem 成员ID项
type UserIdItem struct {
	Userid     string `json:"userid"`
	Department int    `json:"department"`
}

// UserIdList 成员ID列表
type UserIdList struct {
	NextCursor string       `json:"next_cursor"`
	DeptUser   []UserIdItem `json:"dept_user"`
}

// SimpleUser 部门成员简要信息
type SimpleUser struct {
	Userid     string `json:"userid"`
	Name       string `json:"name"`
	Department []int  `json:"department"`
	OpenUserid string `json:"open_userid"`
}

// InviteResult 邀请成员结果
type InviteResult struct {
	InvalidUser  []string `json:"invaliduser"`
	InvalidParty []int    `json:"invalidparty"`
	InvalidTag   []int    `json:"invalidtag"`
}

// userInfoResult 获取成员信息响应
type userInfoResult struct {
	UserInfo
}

// userIdListResult 获取成员ID列表响应
type userIdListResult struct {
	UserIdList
}

// simpleUserListResult 获取部门成员响应
type simpleUserListResult struct {
	UserList []SimpleUser `json:"userlist"`
}

// userDetailListResult 获取部门成员详情响应
type userDetailListResult struct {
	UserList []UserInfo `json:"userlist"`
}

// convertToOpenidResult userid转openid响应
type convertToOpenidResult struct {
	Openid string `json:"openid"`
}

// convertToUseridResult openid转userid响应
type convertToUseridResult struct {
	Userid string `json:"userid"`
}

// inviteResult 邀请成员响应
type inviteResult struct {
	InviteResult
}

// getUseridResult 手机号/邮箱获取userid响应
type getUseridResult struct {
	Userid string `json:"userid"`
}

// ========== 部门管理相关类型 ==========

// CreateDepartmentRequest 创建部门请求
type CreateDepartmentRequest struct {
	Name     string `json:"name"`
	NameEn   string `json:"name_en,omitempty"`
	Parentid int    `json:"parentid"`
	Order    int    `json:"order,omitempty"`
	Id       int    `json:"id,omitempty"`
}

// UpdateDepartmentRequest 更新部门请求
type UpdateDepartmentRequest struct {
	Id       int    `json:"id"`
	Name     string `json:"name,omitempty"`
	NameEn   string `json:"name_en,omitempty"`
	Parentid int    `json:"parentid,omitempty"`
	Order    int    `json:"order,omitempty"`
}

// DepartmentInfo 部门信息
type DepartmentInfo struct {
	Id               int      `json:"id"`
	Name             string   `json:"name"`
	NameEn           string   `json:"name_en"`
	DepartmentLeader []string `json:"department_leader"`
	Parentid         int      `json:"parentid"`
	Order            int      `json:"order"`
}

// DepartmentIdInfo 子部门ID信息
type DepartmentIdInfo struct {
	Id       int `json:"id"`
	Parentid int `json:"parentid"`
	Order    int `json:"order"`
}

// createDepartmentResult 创建部门响应
type createDepartmentResult struct {
	Id int `json:"id"`
}

// departmentListResult 获取部门列表响应
type departmentListResult struct {
	Department []DepartmentInfo `json:"department"`
}

// departmentIdListResult 获取子部门ID列表响应
type departmentIdListResult struct {
	DepartmentId []DepartmentIdInfo `json:"department_id"`
}

// departmentGetResult 获取单个部门详情响应
type departmentGetResult struct {
	Department DepartmentInfo `json:"department"`
}

// ========== 标签管理相关类型 ==========

// TagInfo 标签信息
type TagInfo struct {
	TagId   int    `json:"tagid"`
	TagName string `json:"tagname"`
}

// TagUserInfo 标签成员信息
type TagUserInfo struct {
	Userid string `json:"userid"`
	Name   string `json:"name"`
}

// TagDetail 标签详情（含成员列表）
type TagDetail struct {
	TagName   string        `json:"tagname"`
	UserList  []TagUserInfo `json:"userlist"`
	PartyList []int         `json:"partylist"`
}

// TagMemberResult 标签成员操作结果
type TagMemberResult struct {
	InvalidList  string `json:"invalidlist"`
	InvalidParty []int  `json:"invalidparty"`
}

// createTagResult 创建标签响应
type createTagResult struct {
	TagId int `json:"tagid"`
}

// tagDetailResult 获取标签成员响应
type tagDetailResult struct {
	TagDetail
}

// tagMemberOpResult 标签成员操作响应
type tagMemberOpResult struct {
	TagMemberResult
}

// tagListResult 获取标签列表响应
type tagListResult struct {
	TagList []TagInfo `json:"taglist"`
}

// ========== 异步导入相关类型 ==========

// BatchCallback 异步任务回调信息
type BatchCallback struct {
	Url            string `json:"url"`
	Token          string `json:"token"`
	EncodingAesKey string `json:"encodingaeskey"`
}

// BatchResultItem 异步任务结果项
type BatchResultItem struct {
	Userid  string `json:"userid,omitempty"`
	Action  int    `json:"action"`
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	PartyId int    `json:"partyid,omitempty"`
}

// BatchResult 异步任务结果
type BatchResult struct {
	Status     int               `json:"status"`
	Type       string            `json:"type"`
	Total      int               `json:"total"`
	Percentage int               `json:"percentage"`
	Result     []BatchResultItem `json:"result"`
}

// batchJobResult 批量任务响应
type batchJobResult struct {
	JobId string `json:"jobid"`
}

// batchResultResponse 获取异步任务结果响应
type batchResultResponse struct {
	BatchResult
}

// ========== 异步导出相关类型 ==========

// ExportDataItem 导出文件项
type ExportDataItem struct {
	Url  string `json:"url"`
	Size int    `json:"size"`
	Md5  string `json:"md5"`
}

// ExportResult 导出结果
type ExportResult struct {
	Status   int              `json:"status"`
	DataList []ExportDataItem `json:"data_list"`
}

// exportJobResult 导出任务响应
type exportJobResult struct {
	JobId string `json:"jobid"`
}

// exportResultResponse 获取导出结果响应
type exportResultResponse struct {
	ExportResult
}
