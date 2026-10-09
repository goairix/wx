package official_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/official"
	"github.com/goairix/wx/v2/official/customer"
	"github.com/goairix/wx/v2/official/material"
)

func officialDomainClient(t *testing.T, handler http.HandlerFunc) *official.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/token" {
			_, _ = io.WriteString(w, `{"access_token":"domain-token","expires_in":7200}`)
			return
		}
		if r.URL.Query().Get("access_token") != "domain-token" {
			t.Error("missing shared access token")
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	client, err := official.NewClient(official.Config{AppID: "app", AppSecret: "secret"}, official.WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func assertJSONBody(t *testing.T, r *http.Request, want string) {
	t.Helper()
	var gotJSON, wantJSON interface{}
	if err := json.NewDecoder(r.Body).Decode(&gotJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &wantJSON); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotJSON, wantJSON) {
		t.Errorf("request body = %#v, want %#v", gotJSON, wantJSON)
	}
}

func TestCustomerManagementAndSessionsHTTPContract(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name, method, path, queryKey, queryValue, body, response string
		call                                                     func(*customer.Client) (interface{}, error)
		want                                                     interface{}
	}{
		{name: "add", method: "POST", path: "/customservice/kfaccount/add", body: `{"kf_account":"a@test","nickname":"客服","business_id":"shop"}`, response: `{"errcode":0}`, call: func(c *customer.Client) (interface{}, error) {
			return nil, c.AddAccount(ctx, customer.AccountRequest{Account: "a@test", Nickname: "客服", BusinessID: "shop"})
		}},
		{name: "update", method: "POST", path: "/customservice/kfaccount/update", body: `{"kf_account":"a@test","nickname":"客服"}`, response: `{"errcode":0}`, call: func(c *customer.Client) (interface{}, error) { return nil, c.UpdateAccount(ctx, "a@test", "客服") }},
		{name: "delete", method: "POST", path: "/customservice/kfaccount/del", body: `{"kf_account":"a@test","business_id":"shop"}`, response: `{"errcode":0}`, call: func(c *customer.Client) (interface{}, error) {
			return nil, c.DeleteAccount(ctx, customer.DeleteAccountRequest{Account: "a@test", BusinessID: "shop"})
		}},
		{name: "invite", method: "POST", path: "/customservice/kfaccount/inviteworker", body: `{"kf_account":"a@test","invite_wx":"wx-user"}`, response: `{"errcode":0}`, call: func(c *customer.Client) (interface{}, error) { return nil, c.InviteWorker(ctx, "a@test", "wx-user") }},
		{name: "accounts", method: "GET", path: "/cgi-bin/customservice/getkflist", queryKey: "business_id", queryValue: "shop", response: `{"kf_list":[{"kf_account":"a@test","kf_nick":"客服","kf_id":"1001","kf_headimgurl":"avatar","kf_wx":"wx","invite_wx":"invite","invite_expire_time":123,"invite_status":"waiting","kf_openid":"kf-open"}]}`, call: func(c *customer.Client) (interface{}, error) {
			v, e := c.ListAccounts(ctx, "shop")
			if e != nil {
				return nil, e
			}
			return v[0].ID.String() + ":" + v[0].InviteExpireTime.String() + ":" + v[0].OpenID, nil
		}, want: "1001:123:kf-open"},
		{name: "online", method: "GET", path: "/cgi-bin/customservice/getonlinekflist", response: `{"kf_online_list":[{"kf_account":"a@test","status":1,"kf_id":1001,"accepted_case":2,"kf_openid":"kf-open"}]}`, call: func(c *customer.Client) (interface{}, error) {
			v, e := c.ListOnlineAccounts(ctx, "")
			if e != nil {
				return nil, e
			}
			return v[0].AcceptedCase, nil
		}, want: 2},
		{name: "create session", method: "POST", path: "/customservice/kfsession/create", body: `{"kf_account":"a@test","openid":"open"}`, response: `{"errcode":0}`, call: func(c *customer.Client) (interface{}, error) { return nil, c.CreateSession(ctx, "a@test", "open") }},
		{name: "close session", method: "POST", path: "/customservice/kfsession/close", body: `{"kf_account":"a@test","openid":"open"}`, response: `{"errcode":0}`, call: func(c *customer.Client) (interface{}, error) { return nil, c.CloseSession(ctx, "a@test", "open") }},
		{name: "get session", method: "GET", path: "/customservice/kfsession/getsession", queryKey: "openid", queryValue: "open", response: `{"createtime":123,"kf_account":"a@test"}`, call: func(c *customer.Client) (interface{}, error) { return c.GetSession(ctx, "open") }, want: &customer.SessionStatus{CreateTime: 123, Account: "a@test"}},
		{name: "session list", method: "GET", path: "/customservice/kfsession/getsessionlist", queryKey: "kf_account", queryValue: "a@test", response: `{"sessionlist":[{"createtime":123,"openid":"open"}]}`, call: func(c *customer.Client) (interface{}, error) { return c.ListSessions(ctx, "a@test") }, want: []customer.Session{{CreateTime: 123, OpenID: "open"}}},
		{name: "waiting", method: "GET", path: "/customservice/kfsession/getwaitcase", response: `{"count":1,"waitcaselist":[{"latest_time":123,"openid":"open"}]}`, call: func(c *customer.Client) (interface{}, error) {
			v, e := c.WaitingSessions(ctx)
			if e != nil {
				return nil, e
			}
			return v.Count, nil
		}, want: 1},
		{name: "typing", method: "POST", path: "/cgi-bin/message/custom/typing", body: `{"touser":"open","command":"CancelTyping","businessid":"shop"}`, response: `{"errcode":0}`, call: func(c *customer.Client) (interface{}, error) {
			return nil, c.SetTyping(ctx, customer.TypingRequest{ToUser: "open", Command: customer.CancelTyping, BusinessID: "shop"})
		}},
		{name: "records", method: "POST", path: "/customservice/msgrecord/getmsglist", body: `{"starttime":100,"endtime":200,"msgid":0,"number":100}`, response: `{"recordlist":[{"openid":"open","opercode":2002,"text":"您好","time":123,"worker":"a@test"}],"number":1,"msgid":456}`, call: func(c *customer.Client) (interface{}, error) {
			v, e := c.Records(ctx, customer.RecordRequest{StartTime: 100, EndTime: 200, Number: 100})
			if e != nil {
				return nil, e
			}
			return v.Records[0].Text, nil
		}, want: "您好"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := officialDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != tt.method || r.URL.Path != tt.path {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				if tt.queryKey != "" && r.URL.Query().Get(tt.queryKey) != tt.queryValue {
					t.Errorf("query = %v", r.URL.Query())
				}
				if tt.name == "online" && r.URL.Query().Has("business_id") {
					t.Error("empty optional business_id must be omitted")
				}
				if tt.body != "" {
					assertJSONBody(t, r, tt.body)
				}
				_, _ = io.WriteString(w, tt.response)
			})
			got, err := tt.call(client.CustomerService())
			if err != nil || !reflect.DeepEqual(got, tt.want) || calls != 1 {
				t.Fatalf("result = %#v, want %#v, err = %v, calls = %d", got, tt.want, err, calls)
			}
		})
	}
}

func TestCustomerMessagesHTTPContract(t *testing.T) {
	tests := []struct {
		message customer.Message
		want    string
	}{
		{customer.Message{MsgType: customer.MessageText, Text: &customer.Text{Content: "hello <a>"}}, `"text":{"content":"hello <a>"}`},
		{customer.Message{MsgType: customer.MessageImage, Image: &customer.Media{MediaID: "image"}}, `"image":{"media_id":"image"}`},
		{customer.Message{MsgType: customer.MessageVoice, Voice: &customer.Media{MediaID: "voice"}}, `"voice":{"media_id":"voice"}`},
		{customer.Message{MsgType: customer.MessageVideo, Video: &customer.Video{MediaID: "video", ThumbMediaID: "thumb", Title: "title", Description: "desc"}}, `"video":{"media_id":"video","thumb_media_id":"thumb","title":"title","description":"desc"}`},
		{customer.Message{MsgType: customer.MessageMusic, Music: &customer.Music{Title: "title", Description: "desc", MusicURL: "url", HQMusicURL: "hq", ThumbMediaID: "thumb"}}, `"music":{"title":"title","description":"desc","musicurl":"url","hqmusicurl":"hq","thumb_media_id":"thumb"}`},
		{customer.Message{MsgType: customer.MessageNews, News: &customer.News{Articles: []customer.NewsArticle{{Title: "title", Description: "desc", PicURL: "pic", URL: "url"}}}}, `"news":{"articles":[{"title":"title","description":"desc","picurl":"pic","url":"url"}]}`},
		{customer.Message{MsgType: customer.MessageMPNews, MPNews: &customer.Media{MediaID: "news"}}, `"mpnews":{"media_id":"news"}`},
		{customer.Message{MsgType: customer.MessageMPNewsArticle, MPNewsArticle: &customer.MPNewsArticle{ArticleID: "article"}}, `"mpnewsarticle":{"article_id":"article"}`},
		{customer.Message{MsgType: customer.MessageMenu, Menu: &customer.Menu{HeadContent: "head", List: []customer.MenuItem{{ID: "1", Content: "item"}}, TailContent: "tail"}}, `"msgmenu":{"head_content":"head","list":[{"id":"1","content":"item"}],"tail_content":"tail"}`},
		{customer.Message{MsgType: customer.MessageWXCard, WXCard: &customer.WXCard{CardID: "card"}}, `"wxcard":{"card_id":"card"}`},
		{customer.Message{MsgType: customer.MessageMiniProgramPage, MiniProgramPage: &customer.MiniProgramPage{Title: "title", AppID: "app", PagePath: "pages/index?a=1", ThumbMediaID: "thumb"}}, `"miniprogrampage":{"title":"title","appid":"app","pagepath":"pages/index?a=1","thumb_media_id":"thumb"}`},
		{customer.Message{MsgType: customer.MessageLink, Link: &customer.Link{Title: "title", Description: "desc", URL: "url", ThumbURL: "thumb"}}, `"link":{"title":"title","description":"desc","url":"url","thumb_url":"thumb"}`},
	}
	for _, tt := range tests {
		t.Run(string(tt.message.MsgType), func(t *testing.T) {
			message := tt.message
			message.ToUser = "open"
			message.CustomService = &customer.CustomService{Account: "a@test"}
			message.AIContext = &customer.AIContext{IsAIMessage: 1}
			message.BusinessID = "shop"
			client := officialDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/cgi-bin/message/custom/send" {
					t.Errorf("request = %s %s", r.Method, r.URL.Path)
				}
				assertJSONBody(t, r, `{"touser":"open","msgtype":"`+string(message.MsgType)+`",`+tt.want+`,"customservice":{"kf_account":"a@test"},"aimsgcontext":{"is_ai_msg":1},"businessid":"shop"}`)
				_, _ = io.WriteString(w, `{"errcode":0}`)
			})
			if err := client.CustomerService().Send(context.Background(), message); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOfficialUploadsUseMultipartFilesAndDescription(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name, path, mediaType string
		description           bool
		call                  func(*official.Client) error
	}{
		{"avatar", "/customservice/kfaccount/uploadheadimg", "", false, func(c *official.Client) error {
			return c.CustomerService().UploadHeadImage(ctx, "a@test", "/tmp/avatar.jpg", []byte("file-data"))
		}},
		{"temporary", "/cgi-bin/media/upload", "thumb", false, func(c *official.Client) error {
			v, e := c.Materials().UploadTemporary(ctx, material.TypeThumb, "/tmp/file.jpg", []byte("file-data"))
			if e == nil && (v.MediaID != "media" || v.ThumbMediaID != "thumb" || v.CreatedAt != 123) {
				t.Errorf("upload = %+v", v)
			}
			return e
		}},
		{"permanent", "/cgi-bin/material/add_material", "video", true, func(c *official.Client) error {
			v, e := c.Materials().Add(ctx, material.UploadRequest{Type: material.TypeVideo, Filename: "/tmp/file.mp4", Data: []byte("file-data"), Description: &material.VideoDescription{Title: "标题", Introduction: "简介"}})
			if e == nil && v.MediaID != "media" {
				t.Errorf("upload = %+v", v)
			}
			return e
		}},
		{"article image", "/cgi-bin/media/uploadimg", "", false, func(c *official.Client) error {
			v, e := c.Materials().UploadImage(ctx, "/tmp/file.jpg", []byte("file-data"))
			if e == nil && v.URL != "image-url" {
				t.Errorf("upload = %+v", v)
			}
			return e
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := officialDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != tt.path || r.URL.Query().Get("type") != tt.mediaType {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				if tt.name == "avatar" && r.URL.Query().Get("kf_account") != "a@test" {
					t.Error("missing kf_account query")
				}
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				defer func() { _ = r.MultipartForm.RemoveAll() }()
				file, header, err := r.FormFile("media")
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = file.Close() }()
				data, _ := io.ReadAll(file)
				if string(data) != "file-data" || header.Filename == "/tmp/file.jpg" {
					t.Errorf("file = %q, filename = %q", data, header.Filename)
				}
				if tt.description {
					var v map[string]string
					if err := json.Unmarshal([]byte(r.FormValue("description")), &v); err != nil || v["title"] != "标题" || v["introduction"] != "简介" {
						t.Errorf("description = %q", r.FormValue("description"))
					}
				}
				_, _ = io.WriteString(w, `{"errcode":0,"media_id":"media","thumb_media_id":"thumb","created_at":123,"url":"image-url"}`)
			})
			if err := tt.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestMaterialJSONAndBinaryResponses(t *testing.T) {
	ctx := context.Background()
	client := officialDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/material/get_material" {
			if r.Method != "POST" {
				t.Error("permanent get must POST")
			}
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			switch body["media_id"] {
			case "news":
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"news_item":[{"title":"title","thumb_media_id":"thumb","show_cover_pic":0,"author":"author","digest":"digest","content":"<p>body</p>","url":"url","content_source_url":"source","thumb_url":"thumb-url"}]}`)
				return
			case "video":
				_, _ = io.WriteString(w, `{"title":"title","description":"desc","down_url":"video-url"}`)
				return
			case "error":
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Request-Id", "material-id")
				_, _ = io.WriteString(w, `{"errcode":40007,"errmsg":"invalid media_id"}`)
				return
			}
		} else {
			if r.Method != "GET" || (r.URL.Path != "/cgi-bin/media/get" && r.URL.Path != "/cgi-bin/media/get/jssdk") {
				t.Errorf("request = %s %s", r.Method, r.URL.Path)
			}
			if r.URL.Query().Get("media_id") == "video" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"video_url":"temporary-video"}`)
				return
			}
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G', 0xff})
	})
	m := client.Materials()
	news, err := m.Get(ctx, "news")
	if err != nil || len(news.NewsItem) != 1 || news.NewsItem[0].Content != "<p>body</p>" || news.NewsItem[0].ShowCoverPic != 0 {
		t.Fatalf("news = %+v err=%v", news, err)
	}
	video, err := m.Get(ctx, "video")
	if err != nil || video.DownURL != "video-url" {
		t.Fatalf("video = %+v err=%v", video, err)
	}
	binary, err := m.Get(ctx, "image")
	if err != nil || !bytes.Equal(binary.Data, []byte{0x89, 'P', 'N', 'G', 0xff}) || binary.ContentType != "image/png" {
		t.Fatalf("binary = %+v err=%v", binary, err)
	}
	temp, err := m.GetTemporary(ctx, "video")
	if err != nil || temp.VideoURL != "temporary-video" {
		t.Fatalf("temporary video = %+v err=%v", temp, err)
	}
	for _, get := range []func(context.Context, string) (*material.DownloadResult, error){m.GetTemporary, m.GetHDVoice} {
		v, e := get(ctx, "image")
		if e != nil || !bytes.Equal(v.Data, binary.Data) {
			t.Fatalf("download = %+v err=%v", v, e)
		}
	}
	for _, get := range []func(context.Context, string, io.Writer) (string, error){m.DownloadTo, m.DownloadTemporaryTo, m.DownloadHDVoiceTo} {
		var dst bytes.Buffer
		typ, e := get(ctx, "image", &dst)
		if e != nil || !bytes.Equal(dst.Bytes(), binary.Data) || typ != "image/png" {
			t.Fatalf("stream = %q type=%q err=%v", dst.Bytes(), typ, e)
		}
	}
	_, err = m.Get(ctx, "error")
	var platformErr *wxerrors.Error
	if !errors.As(err, &platformErr) || platformErr.Code != "40007" || platformErr.RequestID != "material-id" {
		t.Fatalf("material error = %+v", err)
	}
	var dst bytes.Buffer
	_, err = m.DownloadTo(ctx, "error", &dst)
	if !errors.As(err, &platformErr) || dst.Len() != 0 {
		t.Fatalf("stream error=%v data=%q", err, dst.Bytes())
	}
}

func TestMaterialListCountDeleteHTTPContract(t *testing.T) {
	ctx := context.Background()
	client := officialDomainClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/material/get_materialcount":
			if r.Method != "GET" {
				t.Error("count must GET")
			}
			_, _ = io.WriteString(w, `{"voice_count":1,"video_count":2,"image_count":3,"news_count":4}`)
		case "/cgi-bin/material/batchget_material":
			if r.Method != "POST" {
				t.Error("list must POST")
			}
			assertJSONBody(t, r, `{"type":"news","offset":0,"count":20}`)
			_, _ = io.WriteString(w, `{"total_count":2,"item_count":1,"item":[{"media_id":"news","update_time":123,"content":{"news_item":[{"title":"title","content":"body"}]}}]}`)
		case "/cgi-bin/material/del_material":
			if r.Method != "POST" {
				t.Error("delete must POST")
			}
			assertJSONBody(t, r, `{"media_id":"media"}`)
			_, _ = io.WriteString(w, `{"errcode":0}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})
	counts, err := client.Materials().Count(ctx)
	if err != nil || counts.NewsCount != 4 || counts.ImageCount != 3 {
		t.Fatalf("count=%+v err=%v", counts, err)
	}
	list, err := client.Materials().List(ctx, material.ListRequest{Type: material.TypeNews, Count: 20})
	if err != nil || list.TotalCount != 2 || list.Items[0].Content.NewsItem[0].Title != "title" {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	if err := client.Materials().Delete(ctx, "media"); err != nil {
		t.Fatal(err)
	}
}

func TestNewDomainsValidationAndCancellation(t *testing.T) {
	client := officialDomainClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid or canceled input reached API") })
	ctx := context.Background()
	for _, message := range []customer.Message{{}, {ToUser: "open", MsgType: customer.MessageText}, {ToUser: "open", MsgType: "unsupported"}} {
		if err := client.CustomerService().Send(ctx, message); err == nil {
			t.Errorf("invalid message accepted: %+v", message)
		}
	}
	if err := client.CustomerService().SetTyping(ctx, customer.TypingRequest{ToUser: "open", Command: "unsupported"}); err == nil {
		t.Error("invalid command accepted")
	}
	if _, err := client.CustomerService().Records(ctx, customer.RecordRequest{StartTime: 100, EndTime: 100 + 86401, Number: 100}); err == nil {
		t.Error("invalid record range accepted")
	}
	if _, err := client.Materials().Add(ctx, material.UploadRequest{Type: material.TypeVideo, Filename: "video.mp4", Data: []byte("video")}); err == nil {
		t.Error("video without description accepted")
	}
	if _, err := client.Materials().List(ctx, material.ListRequest{Type: material.TypeNews, Count: 21}); err == nil {
		t.Error("invalid page size accepted")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := client.CustomerService().Send(canceled, customer.Message{ToUser: "open", MsgType: customer.MessageText, Text: &customer.Text{Content: "hello"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("send cancellation=%v", err)
	}
	if _, err := client.Materials().Get(canceled, "media"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get cancellation=%v", err)
	}
}
