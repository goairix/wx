package customer

import "encoding/json"

// AccountRequest adds a customer service account.
type AccountRequest struct {
	Account    string `json:"kf_account"`
	Nickname   string `json:"nickname"`
	BusinessID string `json:"business_id,omitempty"`
}

// DeleteAccountRequest removes an account, optionally for a submerchant.
type DeleteAccountRequest struct {
	Account    string `json:"kf_account"`
	BusinessID string `json:"business_id,omitempty"`
}

// AccountInfo describes an account and its WeChat binding invitation.
// Number fields accept both numbers and numeric strings returned by WeChat.
type AccountInfo struct {
	Account          string      `json:"kf_account"`
	Nickname         string      `json:"kf_nick"`
	ID               json.Number `json:"kf_id"`
	HeadImageURL     string      `json:"kf_headimgurl"`
	WeChat           string      `json:"kf_wx"`
	InviteWeChat     string      `json:"invite_wx"`
	InviteExpireTime json.Number `json:"invite_expire_time"`
	InviteStatus     string      `json:"invite_status"`
	OpenID           string      `json:"kf_openid"`
}

// OnlineAccount describes an online account and its current workload.
type OnlineAccount struct {
	Account      string      `json:"kf_account"`
	Status       int         `json:"status"`
	ID           json.Number `json:"kf_id"`
	AcceptedCase int         `json:"accepted_case"`
	OpenID       string      `json:"kf_openid"`
}

// SessionStatus is the customer's current assigned session.
type SessionStatus struct {
	CreateTime int64  `json:"createtime"`
	Account    string `json:"kf_account"`
}

// Session is one customer currently served by an account.
type Session struct {
	CreateTime int64  `json:"createtime"`
	OpenID     string `json:"openid"`
}

// WaitingSession is one customer waiting to be served.
type WaitingSession struct {
	LatestTime int64  `json:"latest_time"`
	OpenID     string `json:"openid"`
}

// WaitingSessionList includes the total and up to 100 waiting customers.
type WaitingSessionList struct {
	Count    int              `json:"count"`
	Sessions []WaitingSession `json:"waitcaselist"`
}

// RecordRequest selects a time window and pagination cursor for chat records.
type RecordRequest struct {
	StartTime int64 `json:"starttime"`
	EndTime   int64 `json:"endtime"`
	MessageID int64 `json:"msgid"`
	Number    int   `json:"number"`
}

// Record is one incoming or outgoing customer service chat record.
type Record struct {
	OpenID        string `json:"openid"`
	OperationCode int    `json:"opercode"`
	Text          string `json:"text"`
	Time          int64  `json:"time"`
	Worker        string `json:"worker"`
}

// RecordList includes a page of records and the next message cursor.
type RecordList struct {
	Records   []Record `json:"recordlist"`
	Number    int      `json:"number"`
	MessageID int64    `json:"msgid"`
}
