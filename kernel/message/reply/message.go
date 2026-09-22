package reply

import (
	"encoding/xml"
	"strconv"
	"time"

	"github.com/goairix/wx/v2/kernel/message"
)

// Body 回复消息体
type Body struct {
	XMLName      xml.Name          `xml:"xml"`
	ToUserName   message.CDATAText `xml:"ToUserName"`
	FromUserName message.CDATAText `xml:"FromUserName"`
	CreateTime   string            `xml:"CreateTime"`
	MsgType      message.CDATAText `xml:"MsgType"`

	Content *message.CDATAText `xml:"Content,omitempty"` // 文本消息
	Image   *Media             `xml:"Image,omitempty"`   // 图片消息
	Voice   *Media             `xml:"Voice,omitempty"`   // 音频消息
	Video   *Video             `xml:"Video,omitempty"`   // 视频消息
	Music   *Music             `xml:"Music,omitempty"`   // 音乐消息
}

func (reply *Body) BuildXml(fromUserName, toUserName string) []byte {
	reply.FromUserName = message.Value2CDATA(fromUserName)
	reply.ToUserName = message.Value2CDATA(toUserName)
	reply.CreateTime = strconv.FormatInt(time.Now().Unix(), 10)
	b, _ := xml.MarshalIndent(reply, "", "")
	return b
}

func (reply *Body) ContentType() string {
	return "text/xml"
}

// Media 媒体消息
type Media struct {
	MediaId message.CDATAText `xml:"MediaId,omitempty"`
}

// Music 音乐消息
type Music struct {
	Title        message.CDATAText `xml:"Title"`
	Description  message.CDATAText `xml:"Description"`
	MusicUrl     message.CDATAText `xml:"MusicUrl"`
	HQMusicUrl   message.CDATAText `xml:"HQMusicUrl"`
	ThumbMediaId message.CDATAText `xml:"ThumbMediaId"`
}

// Video 视频消息
type Video struct {
	MediaId     message.CDATAText `xml:"MediaId"`
	Title       message.CDATAText `xml:"Title"`
	Description message.CDATAText `xml:"Description"`
}

// NewText 文本消息
func NewText(content string) *Body {
	return &Body{
		MsgType: message.Value2CDATA("text"),
		Content: message.PtrValue2CDATA(content),
	}
}

// NewImage 图片消息
func NewImage(mediaId string) *Body {
	return &Body{
		MsgType: message.Value2CDATA("image"),
		Image:   &Media{MediaId: message.Value2CDATA(mediaId)},
	}
}

// NewVoice 音频消息
func NewVoice(mediaId string) *Body {
	return &Body{
		MsgType: message.Value2CDATA("voice"),
		Voice:   &Media{MediaId: message.Value2CDATA(mediaId)},
	}
}

// NewVideo 视频消息
func NewVideo(mediaId, title, description string) *Body {
	return &Body{
		MsgType: message.Value2CDATA("video"),
		Video: &Video{
			MediaId:     message.Value2CDATA(mediaId),
			Title:       message.Value2CDATA(title),
			Description: message.Value2CDATA(description),
		},
	}
}

// NewMusic 音乐消息
func NewMusic(title, description, musicUrl, HQMusicUrl, thumbMediaId string) *Body {
	return &Body{
		MsgType: message.Value2CDATA("music"),
		Music: &Music{
			Title:        message.Value2CDATA(title),
			Description:  message.Value2CDATA(description),
			MusicUrl:     message.Value2CDATA(musicUrl),
			HQMusicUrl:   message.Value2CDATA(HQMusicUrl),
			ThumbMediaId: message.Value2CDATA(thumbMediaId),
		},
	}
}
