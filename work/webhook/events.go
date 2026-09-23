package webhook

import "strconv"

// Event contains enterprise message and callback fields. Fields that do not
// apply to a particular event remain at their zero value.
type Event struct {
	Raw    []byte `xml:"-" json:"-"`
	Format string `xml:"-" json:"-"`

	ToUserName   string `xml:"ToUserName" json:"ToUserName"`
	FromUserName string `xml:"FromUserName" json:"FromUserName"`
	CreateTime   int64  `xml:"CreateTime" json:"CreateTime"`
	MessageType  string `xml:"MsgType" json:"MsgType"`
	MessageID    int64  `xml:"MsgId" json:"MsgId"`
	AgentID      int64  `xml:"AgentID" json:"AgentID"`
	Content      string `xml:"Content" json:"Content"`
	MediaID      string `xml:"MediaId" json:"MediaId"`
	PicURL       string `xml:"PicUrl" json:"PicUrl"`
	FormatName   string `xml:"Format" json:"Format"`
	Recognition  string `xml:"Recognition" json:"Recognition"`
	ThumbMediaID string `xml:"ThumbMediaId" json:"ThumbMediaId"`
	LocationX    string `xml:"Location_X" json:"Location_X"`
	LocationY    string `xml:"Location_Y" json:"Location_Y"`
	Scale        string `xml:"Scale" json:"Scale"`
	Label        string `xml:"Label" json:"Label"`
	Title        string `xml:"Title" json:"Title"`
	Description  string `xml:"Description" json:"Description"`
	URL          string `xml:"Url" json:"Url"`

	Event          string          `xml:"Event" json:"Event"`
	EventKey       string          `xml:"EventKey" json:"EventKey"`
	ChangeType     string          `xml:"ChangeType" json:"ChangeType"`
	SuiteID        string          `xml:"SuiteId" json:"SuiteId"`
	SuiteTicket    string          `xml:"SuiteTicket" json:"SuiteTicket"`
	AuthCode       string          `xml:"AuthCode" json:"AuthCode"`
	TimeStamp      int64           `xml:"TimeStamp" json:"TimeStamp"`
	UserID         string          `xml:"UserID" json:"UserID"`
	NewUserID      string          `xml:"NewUserID" json:"NewUserID"`
	Name           string          `xml:"Name" json:"Name"`
	Department     string          `xml:"Department" json:"Department"`
	MainDepartment string          `xml:"MainDepartment" json:"MainDepartment"`
	IsLeaderInDept string          `xml:"IsLeaderInDept" json:"IsLeaderInDept"`
	DirectLeader   string          `xml:"DirectLeader" json:"DirectLeader"`
	Position       string          `xml:"Position" json:"Position"`
	Mobile         string          `xml:"Mobile" json:"Mobile"`
	Gender         string          `xml:"Gender" json:"Gender"`
	Email          string          `xml:"Email" json:"Email"`
	BizMail        string          `xml:"BizMail" json:"BizMail"`
	Status         string          `xml:"Status" json:"Status"`
	Avatar         string          `xml:"Avatar" json:"Avatar"`
	Alias          string          `xml:"Alias" json:"Alias"`
	Telephone      string          `xml:"Telephone" json:"Telephone"`
	Address        string          `xml:"Address" json:"Address"`
	ExtAttr        *ContactExtAttr `xml:"ExtAttr" json:"ExtAttr"`
	PartyID        string          `xml:"Id" json:"Id"`
	ParentID       string          `xml:"ParentId" json:"ParentId"`
	Order          string          `xml:"Order" json:"Order"`
	TagID          string          `xml:"TagId" json:"TagId"`
	AddUserItems   string          `xml:"AddUserItems" json:"AddUserItems"`
	DelUserItems   string          `xml:"DelUserItems" json:"DelUserItems"`
	AddPartyItems  string          `xml:"AddPartyItems" json:"AddPartyItems"`
	DelPartyItems  string          `xml:"DelPartyItems" json:"DelPartyItems"`

	ExternalUserID    string `xml:"ExternalUserID" json:"ExternalUserID"`
	State             string `xml:"State" json:"State"`
	WelcomeCode       string `xml:"WelcomeCode" json:"WelcomeCode"`
	Source            string `xml:"Source" json:"Source"`
	FailReason        string `xml:"FailReason" json:"FailReason"`
	ChatID            string `xml:"ChatId" json:"ChatId"`
	UpdateDetail      string `xml:"UpdateDetail" json:"UpdateDetail"`
	JoinScene         int    `xml:"JoinScene" json:"JoinScene"`
	QuitScene         int    `xml:"QuitScene" json:"QuitScene"`
	MemberChangeCount int    `xml:"MemChangeCnt" json:"MemChangeCnt"`
	TagType           string `xml:"TagType" json:"TagType"`

	TaskID        string        `xml:"TaskId" json:"TaskId"`
	CardType      string        `xml:"CardType" json:"CardType"`
	ResponseCode  string        `xml:"ResponseCode" json:"ResponseCode"`
	SelectedItems SelectedItems `xml:"SelectedItems" json:"SelectedItems"`
	LivingID      string        `xml:"LivingId" json:"LivingId"`
	BatchJob      BatchJob      `xml:"BatchJob" json:"BatchJob"`
	ApprovalInfo  ApprovalInfo  `xml:"ApprovalInfo" json:"ApprovalInfo"`
}

// ContactExtAttr contains extended member attributes.
type ContactExtAttr struct {
	Items []ContactExtAttrItem `xml:"Item" json:"Item"`
}

// ContactExtAttrItem is one member extension attribute.
type ContactExtAttrItem struct {
	Name string `xml:"Name" json:"Name"`
	Type string `xml:"Type" json:"Type"`
	Text struct {
		Value string `xml:"Value" json:"Value"`
	} `xml:"Text" json:"Text"`
	Web struct {
		Title string `xml:"Title" json:"Title"`
		URL   string `xml:"Url" json:"Url"`
	} `xml:"Web" json:"Web"`
}

// BatchJob describes an asynchronous contact job result.
type BatchJob struct {
	JobID   string `xml:"JobId" json:"JobId"`
	JobType string `xml:"JobType" json:"JobType"`
	ErrCode int    `xml:"ErrCode" json:"ErrCode"`
	ErrMsg  string `xml:"ErrMsg" json:"ErrMsg"`
}

// SelectedItems contains template card selections.
type SelectedItems struct {
	Items []SelectedItem `xml:"SelectedItem" json:"SelectedItem"`
}

// SelectedItem is one template card answer.
type SelectedItem struct {
	QuestionKey string `xml:"QuestionKey" json:"QuestionKey"`
	OptionIDs   struct {
		IDs []string `xml:"OptionId" json:"OptionId"`
	} `xml:"OptionIds" json:"OptionIds"`
}

// ApprovalInfo describes an approval status change.
type ApprovalInfo struct {
	Number       string           `xml:"SpNo" json:"SpNo"`
	Name         string           `xml:"SpName" json:"SpName"`
	Status       int              `xml:"SpStatus" json:"SpStatus"`
	TemplateID   string           `xml:"TemplateId" json:"TemplateId"`
	ApplyTime    int64            `xml:"ApplyTime" json:"ApplyTime"`
	Applicant    ApprovalUser     `xml:"Applyer" json:"Applyer"`
	Records      []ApprovalRecord `xml:"SpRecord" json:"SpRecord"`
	Notifiers    []ApprovalUser   `xml:"Notifyer" json:"Notifyer"`
	StatusChange int              `xml:"StatuChangeEvent" json:"StatuChangeEvent"`
}

// ApprovalUser identifies an approval participant.
type ApprovalUser struct {
	UserID string `xml:"UserId" json:"UserId"`
	Party  string `xml:"Party" json:"Party"`
}

// ApprovalRecord is one approval stage.
type ApprovalRecord struct {
	Status       int              `xml:"SpStatus" json:"SpStatus"`
	ApproverAttr int              `xml:"ApproverAttr" json:"ApproverAttr"`
	Details      []ApprovalDetail `xml:"Details" json:"Details"`
}

// ApprovalDetail is one approver's decision.
type ApprovalDetail struct {
	Approver ApprovalUser `xml:"Approver" json:"Approver"`
	Speech   string       `xml:"Speech" json:"Speech"`
	Status   int          `xml:"SpStatus" json:"SpStatus"`
	Time     int64        `xml:"SpTime" json:"SpTime"`
}

// SuiteTicketEvent represents a third-party suite ticket callback.
type SuiteTicketEvent struct{ Event }

// AuthorizationEvent represents authorization lifecycle callbacks.
type AuthorizationEvent struct{ Event }

// ContactChangeEvent represents member, department, and tag changes.
type ContactChangeEvent struct{ Event }

// ExternalContactChangeEvent represents external contact lifecycle changes.
type ExternalContactChangeEvent struct{ Event }

// GroupChatChangeEvent represents customer group chat lifecycle changes.
type GroupChatChangeEvent struct{ Event }

// BatchJobEvent represents completion of an asynchronous contact job.
type BatchJobEvent struct{ Event }

// ExternalTagChangeEvent represents enterprise customer tag changes.
type ExternalTagChangeEvent struct{ Event }

// TemplateCardEvent represents template card interactions.
type TemplateCardEvent struct{ Event }

// LivingStatusChangeEvent represents live stream status changes.
type LivingStatusChangeEvent struct {
	Event
	Status int
}

// ApprovalEvent represents approval status changes.
type ApprovalEvent struct{ Event }

// SuiteTicket returns the event as a suite ticket event.
func (e Event) SuiteTicketEvent() SuiteTicketEvent { return SuiteTicketEvent{Event: e} }

// Authorization returns the event as an authorization event.
func (e Event) Authorization() AuthorizationEvent { return AuthorizationEvent{Event: e} }

// ContactChange returns the event as a contact change event.
func (e Event) ContactChange() ContactChangeEvent { return ContactChangeEvent{Event: e} }

// ExternalContactChange returns the event as an external contact event.
func (e Event) ExternalContactChange() ExternalContactChangeEvent {
	return ExternalContactChangeEvent{Event: e}
}

// GroupChatChange returns the event as a customer group event.
func (e Event) GroupChatChange() GroupChatChangeEvent {
	return GroupChatChangeEvent{Event: e}
}

// BatchJobResult returns the event as an asynchronous job event.
func (e Event) BatchJobResult() BatchJobEvent { return BatchJobEvent{Event: e} }

// ExternalTagChange returns the event as an enterprise tag event.
func (e Event) ExternalTagChange() ExternalTagChangeEvent {
	return ExternalTagChangeEvent{Event: e}
}

// TemplateCard returns the event as a template card event.
func (e Event) TemplateCard() TemplateCardEvent { return TemplateCardEvent{Event: e} }

// LivingStatusChange returns the event as a live stream event.
func (e Event) LivingStatusChange() LivingStatusChangeEvent {
	status, _ := strconv.Atoi(e.Status)
	return LivingStatusChangeEvent{Event: e, Status: status}
}

// Approval returns the event as an approval event.
func (e Event) Approval() ApprovalEvent { return ApprovalEvent{Event: e} }
