package user

// Info is a user profile returned by cgi-bin/user/info.
type Info struct {
	Subscribe      uint8   `json:"subscribe"`
	Openid         string  `json:"openid"`
	UnionID        string  `json:"unionid"`
	Language       string  `json:"language"`
	Remark         string  `json:"remark"`
	GroupID        int64   `json:"groupid"`
	TagIdList      []int64 `json:"tagid_list"`
	SubscribeTime  int64   `json:"subscribe_time"`
	SubscribeScene string  `json:"subscribe_scene"`
	QrScene        int64   `json:"qr_scene"`
	QrSceneStr     string  `json:"qr_scene_str"`
	Nickname       string  `json:"nickname"`
	Sex            uint8   `json:"sex"`
	Province       string  `json:"province"`
	City           string  `json:"city"`
	Country        string  `json:"country"`
	HeadImgURL     string  `json:"headimgurl"`
}
