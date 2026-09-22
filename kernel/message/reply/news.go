package reply

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/goairix/wx/v2/kernel/message"
)

// News 图文消息
type News struct {
	XMLName      xml.Name          `xml:"xml"`
	ToUserName   message.CDATAText `xml:"ToUserName"`
	FromUserName message.CDATAText `xml:"FromUserName"`
	CreateTime   string            `xml:"CreateTime"`
	MsgType      message.CDATAText `xml:"MsgType"`
	ArticleCount int               `xml:"ArticleCount"`
	Articles     []NewsArticle     `xml:"Articles"`
}

// NewsArticle 图文消息文章
type NewsArticle struct {
	Item struct {
		Title       message.CDATAText `xml:"Title"`
		Description message.CDATAText `xml:"Description"`
		PicUrl      message.CDATAText `xml:"PicUrl"`
		Url         message.CDATAText `xml:"Url"`
	} `xml:"item"`
}

func (reply *News) BuildXml(fromUserName, toUserName string) []byte {
	reply.FromUserName = message.Value2CDATA(fromUserName)
	reply.ToUserName = message.Value2CDATA(toUserName)
	reply.CreateTime = strconv.FormatInt(time.Now().Unix(), 10)
	b, _ := xml.MarshalIndent(reply, "", "")
	return b
}

func (reply *News) ContentType() string {
	return "text/xml"
}

// NewNews 图文消息
func NewNews(articles []map[string]string) *News {
	length := len(articles)
	list := make([]NewsArticle, length)
	for i, article := range articles {
		art := NewsArticle{}
		for key, value := range article {
			switch key {
			case "title":
				art.Item.Title = message.Value2CDATA(value)
				break
			case "description":
				art.Item.Description = message.Value2CDATA(value)
				break
			case "pic_url":
				art.Item.PicUrl = message.Value2CDATA(value)
				break
			case "url":
				art.Item.Url = message.Value2CDATA(value)
				break
			}
		}
		list[i] = art
	}
	return &News{
		MsgType:      message.Value2CDATA("news"),
		ArticleCount: length,
		Articles:     list,
	}
}
