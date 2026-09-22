package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"time"

	"github.com/goairix/wx/health_card"
	"github.com/goairix/wx/kernel/contracts"
	"github.com/goairix/wx/kernel/event"
	"github.com/goairix/wx/kernel/message"
	"github.com/goairix/wx/kernel/message/reply"
	"github.com/goairix/wx/mini_program"
	"github.com/goairix/wx/mini_program/platform/multi_terminal"
	"github.com/goairix/wx/official"
	"github.com/goairix/wx/official/menu"
	templateMessage "github.com/goairix/wx/official/message"
	"github.com/goairix/wx/open_platform"
	"github.com/goairix/wx/open_platform/authorizer"
	"github.com/goairix/wx/support/cache"
)

var (
	openPlatform  *open_platform.OpenPlatform
	officialSdk   *official.Official
	miniProgram   *mini_program.MiniProgram
	multiTerminal *multi_terminal.MultiTerminal
	healthCard    *health_card.Client
	cacheItem     cache.Cache
)

func init() {
	cacheItem = cache.NewMemoryCache()
	// openPlatform = open_platform.New(
	//	"wx1fd773ad1f20c04e",
	//	"6a3079cc28f3f7efc0be7b4e719c5e35",
	//	"dysodeng",
	//	"uDb99HRspPGSyu6V2JcykyQArDzXJz4WTCTPFqXM0cr",
	//	open_platform.WithCache(cacheItem),
	// )
	// officialSdk = official.New(
	//	"wx5542a6f077ff70c6",
	//	"11b1da13cb3eb27626ec1e42f035f467",
	//	"dysodeng",
	//	"uDb99HRspPGSyu6V2JcykyQArDzXJz4WTCTPFqXM0cr",
	//	official.WithCache(cacheItem),
	// )
	miniProgram = mini_program.New(
		"wxcdeba3fc090d79a0",
		"38341520b87d04c71ecc4231aa98e6bd",
		"",
		"",
		mini_program.WithCache(cacheItem),
	)

	multiTerminal = multi_terminal.New(
		"wx4233b234a9f956c2",
		"ec06ce51cf29ddd5be450a5f1e402c0f",
		"",
		"",
		multi_terminal.WithCache(cacheItem),
	)

	healthCard = health_card.New(
		"2a52d1ac06374477a7a0df8fdf093f43",
		"9d0ebd1071024304a0a40c8b91d5b302",
		"41485",
	)

}

func main() {
	// 健康卡接口通过领域入口调用，例如：healthCard.Patient().GetRegistrationInfo(...)
	_ = healthCard.Patient()
}

func mains() {

	openPlatformServer := openPlatform.Server()

	appServer := officialSdk.Server()
	appServer.On(func(ctx context.Context, account contracts.AccountInterface, messageBody *message.Message) (*reply.Reply, error) {
		log.Println("这里是用户自定义的消息处理器")
		log.Println(messageBody)
		header := messageBody.Header()
		switch header.MsgType {
		case "text":
			text := messageBody.Text()
			if text.Content == "openid" {
				return reply.NewReply(reply.NewText(header.FromUserName)), nil
			}
		case "event":
			e := messageBody.EventMessage()
			switch strings.ToLower(e.Event) {
			case "click":
				m := e.Menu()
				if m.EventKey == "openid" {
					return reply.NewReply(reply.NewText(header.FromUserName)), nil
				}
				if m.EventKey == "click_menu:1" {
					return reply.NewReply(reply.NewText("你真帅")), nil
				}
			}
		}
		return reply.NewReply(reply.NewText("你好呀")), nil
	}, event.All)

	h := http.DefaultServeMux

	h.HandleFunc("/qSbQCJyd5Q.txt", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("6e884fadb9c3257016d530fc8293f3fc"))
	})
	h.HandleFunc("/wx/open/event", func(writer http.ResponseWriter, request *http.Request) {
		log.Println("event")
		openPlatformServer.Serve(request, writer)
	})
	h.HandleFunc("/wx/open/auth", func(writer http.ResponseWriter, request *http.Request) {
		authUrl, err := openPlatform.Authorizer().PreAuthorizationUrl("http://wx.dysodeng.com/wx/open/auth/callback", authorizer.AuthAll)
		if err != nil {
			_, _ = writer.Write([]byte(err.Error()))
		} else {
			tpl := template.New("auth")
			t, _ := tpl.Parse(fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="referrer" content="origin">
    <title>授权跳转</title>
</head>
<body>
</body>
<script>
	const gotoLink = document.createElement('a');
    gotoLink.href = '%s';
    document.body.appendChild(gotoLink);
    gotoLink.click();
</script>
</html>
`, authUrl))
			_ = t.Execute(writer, "")
		}
	})
	h.HandleFunc("/wx/open/auth/h5", func(writer http.ResponseWriter, request *http.Request) {
		authUrl, err := openPlatform.Authorizer().MobilePreAuthorizationUrl("http://wx.dysodeng.com/wx/open/auth/callback", authorizer.AuthAll)
		if err != nil {
			_, _ = writer.Write([]byte(err.Error()))
		} else {
			tpl := template.New("auth")
			t, _ := tpl.Parse(fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="referrer" content="origin">
    <title>授权跳转</title>
</head>
<body>
</body>
<script>
	const gotoLink = document.createElement('a');
    gotoLink.href = '%s';
    document.body.appendChild(gotoLink);
    gotoLink.click();
</script>
</html>
`, authUrl))
			_ = t.Execute(writer, "")
		}
	})
	h.HandleFunc("/wx/open/auth/callback", func(writer http.ResponseWriter, request *http.Request) {
		authCode := request.URL.Query().Get("auth_code")
		log.Printf("auth_code:%s", authCode)
		info, _ := openPlatform.Authorizer().AuthorizationInfo(authCode)
		log.Printf("%+v", info)
		_, _ = writer.Write([]byte(fmt.Sprintf("auth_code:%s", authCode)))
	})

	h.HandleFunc("/wx/open/jssdk/config", func(writer http.ResponseWriter, request *http.Request) {
		o := openPlatform.Official("wx798e1b0582e22401", "refreshtoken@@@FBFqHwrqfM2Xvgm5MRAeK-J2K5oohfApcEDuqQO-WmA")
		config := o.Jssdk().SetUrl("http://wx.dysodeng.com/wx/open/jssdk").BuildConfig(
			[]string{"updateAppMessageShareData", "updateTimelineShareData"}, true, true)
		cb, _ := json.Marshal(config)
		writer.Header().Set("content-type", "application/json")
		_, _ = writer.Write(cb)
	})

	h.HandleFunc("/wx/mp/session", func(writer http.ResponseWriter, request *http.Request) {
		code := ""
		log.Println(miniProgram.Auth().Session(code))
	})

	h.HandleFunc("/wx/event", func(w http.ResponseWriter, r *http.Request) {
		appServer.Serve(r, w)
	})
	h.HandleFunc("/wx/user", func(writer http.ResponseWriter, request *http.Request) {
		err := officialSdk.Menu().Create([]menu.Item{
			{
				Type: "click",
				Name: "点我",
				Key:  "click_menu:1",
			},
			{
				Name: "多级菜单",
				SubButton: []menu.Item{
					{
						Type: "view",
						Name: "百度",
						Url:  "https://www.baidu.com",
					},
					{
						Type: "pic_sysphoto",
						Name: "拍照",
						Key:  "photo:1",
					},
					{
						Type: "click",
						Name: "OpenID",
						Key:  "openid",
					},
				},
			},
		})
		if err != nil {
			log.Printf("%+v", err)
		} else {
			// log.Printf("%+v", l)
		}
	})
	h.HandleFunc("/wx/menu", func(writer http.ResponseWriter, request *http.Request) {
		l, err := officialSdk.Menu().Info()
		if err != nil {
			log.Printf("%+v", err)
		} else {
			j, _ := json.Marshal(l)
			log.Println(string(j))
		}
	})
	h.HandleFunc("/wx/grant", func(writer http.ResponseWriter, request *http.Request) {
		officialSdk.OAuth().
			WithScope("snsapi_userinfo").
			WithRedirectUrl("http://wx.dysodeng.com/wx/grant/callback").Redirect(writer, request)
	})
	h.HandleFunc("/wx/grant/callback", func(writer http.ResponseWriter, request *http.Request) {
		code := request.URL.Query().Get("code")
		if code != "" {
			u, err := officialSdk.OAuth().UserFromCode(code)
			if err != nil {
				_, _ = writer.Write([]byte(err.Error()))
			} else {
				b, _ := json.Marshal(u)
				_, _ = writer.Write(b)
			}
		}
	})
	h.HandleFunc("/wx/message", func(writer http.ResponseWriter, request *http.Request) {
		openid := request.URL.Query().Get("openid")
		if openid != "" {
			msgId, err := officialSdk.TemplateMessage().Send(templateMessage.Message{
				TemplateId: "tUD-OvPFZsahGjSjJBHjLfGgL-ZEiSmHY3aVA1XR2Zo",
				ToUser:     openid,
				Data: map[string]*templateMessage.DataValue{
					"first": {
						Value: "恭喜你中了大奖了",
					},
					"keyword1": {
						Value: "没错，就是你",
					},
					"keyword2": {
						Value: time.Now().Format("2006-01-02 03:04:05"),
					},
					"remark": {
						Value: "请立即去元宇宙中领取",
					},
				},
			})
			if err != nil {
				_, _ = writer.Write([]byte(err.Error()))
			} else {
				_, _ = writer.Write([]byte(fmt.Sprintf("发送成功 msgId: %d", msgId)))
			}
		}
	})
	h.HandleFunc("/wx/jssdk/config", func(writer http.ResponseWriter, request *http.Request) {
		config := officialSdk.Jssdk().SetUrl("http://wx.dysodeng.com/wx/jssdk").BuildConfig(
			[]string{"updateAppMessageShareData", "updateTimelineShareData"}, true, true)
		cb, _ := json.Marshal(config)
		_, _ = writer.Write(cb)
	})
	h.HandleFunc("/wx/jssdk", func(writer http.ResponseWriter, request *http.Request) {
		tpl := template.New("jssdk")
		t, _ := tpl.Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>微信jssdk</title>
<script src="https://res.wx.qq.com/open/js/jweixin-1.6.0.js"></script>
<script src="https://cdn.bootcdn.net/ajax/libs/jquery/3.6.0/jquery.min.js"></script>
</head>
<body>

<script>
    $(function () {
        $.ajax({
            url: 'http://wx.dysodeng.com/wx/jssdk/config',
            type: 'get',
            success: function (res) {
                var config = JSON.parse(res)
                wx.config(config);
            }
        });
    })
</script>
</body>
</html>
`)
		_ = t.Execute(writer, "")
	})
	h.HandleFunc("/wx/telephone", func(writer http.ResponseWriter, request *http.Request) {
		phone, err := miniProgram.User().GetPhoneNumber("63172580ff56db79c77787534e086f354ca920a6dc84e1f42951479e97870063", "")
		if err != nil {
			log.Printf("%+v", err)
			return
		}
		log.Printf("%+v", phone)
	})

	server := http.Server{
		Addr:    ":9010",
		Handler: h,
	}

	log.Println("http server listening to :9010")

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("%+v\n", err)
	}
}
