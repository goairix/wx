package agent

// Summary identifies an application accessible to the current credential.
type Summary struct {
	AgentID       int64  `json:"agentid"`
	Name          string `json:"name"`
	SquareLogoURL string `json:"square_logo_url"`
}

// Detail contains application settings and visibility scopes.
type Detail struct {
	Summary
	Description             string             `json:"description"`
	AllowUserInfos          AllowedUserInfos   `json:"allow_userinfos"`
	AllowPartys             AllowedDepartments `json:"allow_partys"`
	AllowTags               AllowedTags        `json:"allow_tags"`
	Close                   int                `json:"close"`
	RedirectDomain          string             `json:"redirect_domain"`
	ReportLocationFlag      int                `json:"report_location_flag"`
	IsReportEnter           int                `json:"isreportenter"`
	HomeURL                 string             `json:"home_url"`
	CustomizedPublishStatus int                `json:"customized_publish_status,omitempty"`
}

// AllowedUserInfos contains members in the application's visibility scope.
type AllowedUserInfos struct {
	User []AllowedUser `json:"user"`
}

// AllowedUser identifies one visible member.
type AllowedUser struct {
	UserID string `json:"userid"`
}

// AllowedDepartments contains visible department IDs.
type AllowedDepartments struct {
	PartyID []int64 `json:"partyid"`
}

// AllowedTags contains visible tag IDs.
type AllowedTags struct {
	TagID []int64 `json:"tagid"`
}

// SetRequest updates application settings. Nil fields are omitted; pointers
// preserve explicit empty strings and zero flags.
type SetRequest struct {
	AgentID            int64   `json:"agentid"`
	ReportLocationFlag *int    `json:"report_location_flag,omitempty"`
	LogoMediaID        *string `json:"logo_mediaid,omitempty"`
	Name               *string `json:"name,omitempty"`
	Description        *string `json:"description,omitempty"`
	RedirectDomain     *string `json:"redirect_domain,omitempty"`
	IsReportEnter      *int    `json:"isreportenter,omitempty"`
	HomeURL            *string `json:"home_url,omitempty"`
}

// Menu contains an application's custom menu.
type Menu struct {
	Button []MenuButton `json:"button"`
}

// MenuButton describes an action or a parent containing submenu buttons.
type MenuButton struct {
	Type      string       `json:"type,omitempty"`
	Name      string       `json:"name"`
	Key       string       `json:"key,omitempty"`
	URL       string       `json:"url,omitempty"`
	AppID     string       `json:"appid,omitempty"`
	PagePath  string       `json:"pagepath,omitempty"`
	SubButton []MenuButton `json:"sub_button,omitempty"`
}

// WorkbenchType identifies an application's workbench display style.
type WorkbenchType string

const (
	// WorkbenchNormalType disables custom workbench display.
	WorkbenchNormalType WorkbenchType = "normal"
	// WorkbenchKeyDataType displays up to four key metrics.
	WorkbenchKeyDataType WorkbenchType = "keydata"
	// WorkbenchImageType displays an image.
	WorkbenchImageType WorkbenchType = "image"
	// WorkbenchListType displays up to three list items.
	WorkbenchListType WorkbenchType = "list"
	// WorkbenchWebviewType displays an embedded web page.
	WorkbenchWebviewType WorkbenchType = "webview"
)

// WorkbenchData contains the payload matching Type. Normal is valid only when
// setting the application template; user data requires a custom display type.
type WorkbenchData struct {
	Type    WorkbenchType     `json:"type"`
	KeyData *WorkbenchKeyData `json:"keydata,omitempty"`
	Image   *WorkbenchImage   `json:"image,omitempty"`
	List    *WorkbenchList    `json:"list,omitempty"`
	Webview *WorkbenchWebview `json:"webview,omitempty"`
}

// WorkbenchKeyData contains key metrics.
type WorkbenchKeyData struct {
	Items []WorkbenchKeyDataItem `json:"items"`
}

// WorkbenchKeyDataItem is one metric and its optional navigation target.
type WorkbenchKeyDataItem struct {
	Key      string `json:"key,omitempty"`
	Data     string `json:"data"`
	JumpURL  string `json:"jump_url,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

// WorkbenchImage is a workbench image and its optional navigation target.
type WorkbenchImage struct {
	URL      string `json:"url"`
	JumpURL  string `json:"jump_url,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

// WorkbenchList contains workbench list items.
type WorkbenchList struct {
	Items []WorkbenchListItem `json:"items"`
}

// WorkbenchListItem is one list entry and its optional navigation target.
type WorkbenchListItem struct {
	Title    string `json:"title"`
	JumpURL  string `json:"jump_url,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

// WorkbenchWebview configures an embedded page. Height accepts single_row or
// double_row; enabling in-page clicks makes JumpURL ineffective.
type WorkbenchWebview struct {
	URL                string `json:"url"`
	JumpURL            string `json:"jump_url,omitempty"`
	PagePath           string `json:"pagepath,omitempty"`
	Height             string `json:"height,omitempty"`
	HideTitle          bool   `json:"hide_title,omitempty"`
	EnableWebviewClick bool   `json:"enable_webview_click,omitempty"`
}

// WorkbenchTemplate contains the display style and optional enterprise defaults.
type WorkbenchTemplate struct {
	WorkbenchData
	ReplaceUserData bool `json:"replace_user_data,omitempty"`
}

// SetWorkbenchTemplateRequest configures the application template. Setting
// ReplaceUserData overrides existing workbench data for all application users.
type SetWorkbenchTemplateRequest struct {
	AgentID int64 `json:"agentid"`
	WorkbenchTemplate
}

// SetWorkbenchDataRequest updates workbench data for one visible member.
type SetWorkbenchDataRequest struct {
	AgentID int64  `json:"agentid"`
	UserID  string `json:"userid"`
	WorkbenchData
}

// BatchSetWorkbenchDataRequest updates identical data for up to 1000 members.
type BatchSetWorkbenchDataRequest struct {
	AgentID    int64         `json:"agentid"`
	UserIDList []string      `json:"userid_list"`
	Data       WorkbenchData `json:"data"`
}
