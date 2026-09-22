package encryptor

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/goairix/wx/v2/kernel/message"
)

const (
	ErrInvalidSignature = -40001 // Signature verification failed
	ErrParseXml         = -40002 // Parse XML failed
	ErrCalcSignature    = -40003 // Calculating the signature failed
	ErrInvalidAesKey    = -40004 // Invalid AESKey
	ErrInvalidAppID     = -40005 // Check AppID failed
	ErrEncryptAes       = -40006 // AES EncryptionInterface failed
	ErrDecryptAes       = -40007 // AES decryption failed
	ErrInvalidXml       = -40008 // Invalid XML
	ErrBase64Encode     = -40009 // Base64 encoding failed
	ErrBase64Decode     = -40010 // Base64 decoding failed
	ErrXmlBuild         = -40011 // XML build failed
	ErrIllegalBuffer    = -41003 // Illegal buffer
)

// TextResponseBody 文本回复消息
type TextResponseBody struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   CDATAText
	FromUserName CDATAText
	CreateTime   string
	MsgType      CDATAText
	Content      CDATAText
}

// EncryptRequestBody 加密消息
type EncryptRequestBody struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string
	Encrypt    string
}

// EncryptResponseBody 加密回复消息
type EncryptResponseBody struct {
	XMLName      xml.Name `xml:"xml"`
	Encrypt      CDATAText
	MsgSignature CDATAText
	TimeStamp    string
	Nonce        CDATAText
}

// CDATAText 文本域
type CDATAText struct {
	Text string `xml:",innerxml"`
}

type Encryptor struct {
	appId  string
	token  string
	aesKey string
}

func NewEncryptor(appId, token, aesKey string) *Encryptor {
	aesKeyByte, _ := base64.StdEncoding.DecodeString(aesKey + "=")
	return &Encryptor{
		appId:  appId,
		token:  token,
		aesKey: string(aesKeyByte),
	}
}

// makeSignature 生成签名串
func (e *Encryptor) makeSignature(timestamp, nonce string) string {
	sl := []string{e.token, timestamp, nonce}
	sort.Strings(sl)
	s := sha1.New()
	_, _ = io.WriteString(s, strings.Join(sl, ""))
	return fmt.Sprintf("%x", s.Sum(nil))
}

// makeMsgSignature 生成消息签名串
func (e *Encryptor) makeMsgSignature(timestamp, nonce, msgEncrypt string) string {
	sl := []string{e.token, timestamp, nonce, msgEncrypt}
	sort.Strings(sl)
	s := sha1.New()
	_, _ = io.WriteString(s, strings.Join(sl, ""))
	return fmt.Sprintf("%x", s.Sum(nil))
}

// ValidSignature 验证Url签名
func (e *Encryptor) ValidSignature(timestamp, nonce, signature string) bool {
	return e.makeSignature(timestamp, nonce) == signature
}

// ValidMsgSignature 验证消息签名
func (e *Encryptor) ValidMsgSignature(timestamp, nonce, msgEncrypt, signature string) bool {
	return e.makeMsgSignature(timestamp, nonce, msgEncrypt) == signature
}

// ParseEncryptBody 解析加密消息数据
func (e *Encryptor) ParseEncryptBody(r *http.Request) (*EncryptRequestBody, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	requestBody := &EncryptRequestBody{}
	_ = xml.Unmarshal(body, requestBody)

	return requestBody, nil
}

// ParseTextBody 解析文本消息数据
func (e *Encryptor) ParseTextBody(r *http.Request) (*message.Message, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	messageBody := &message.Message{}
	_ = xml.Unmarshal(body, messageBody)
	messageBody.RawBody = body

	return messageBody, nil
}

// MakeEncryptBody 构建加密消息体
func (e *Encryptor) MakeEncryptBody(xmlBody []byte, timestamp, nonce string) ([]byte, error) {
	encryptBody := &EncryptResponseBody{}

	encryptXmlData, _ := e.makeEncryptXmlData(xmlBody)
	encryptBody.Encrypt = value2CDATA(encryptXmlData)
	encryptBody.MsgSignature = value2CDATA(e.makeMsgSignature(timestamp, nonce, encryptXmlData))
	encryptBody.TimeStamp = strconv.FormatInt(time.Now().Unix(), 10)
	encryptBody.Nonce = value2CDATA(nonce)

	return xml.MarshalIndent(encryptBody, "", "")
}

// makeEncryptXmlData 构建加密消息XML
func (e *Encryptor) makeEncryptXmlData(xmlBody []byte) (string, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, int32(len(xmlBody)))
	if err != nil {
		fmt.Println("Binary write err:", err)
	}
	bodyLength := buf.Bytes()

	randomBytes := []byte("abcdefghijklmnop")

	plainData := bytes.Join([][]byte{randomBytes, bodyLength, xmlBody, []byte(e.appId)}, nil)
	cipherData, err := e.AesEncrypt(plainData)
	if err != nil {
		return "", errors.New("aesEncrypt error")
	}

	return base64.StdEncoding.EncodeToString(cipherData), nil
}

// AesEncrypt 加密
func (e *Encryptor) AesEncrypt(plainData []byte) ([]byte, error) {
	aesKey := []byte(e.aesKey)
	k := len(aesKey)
	if len(plainData)%k != 0 {
		plainData = PKCS7Pad(plainData, k)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	// IV 为 AESKey 的前16字节
	iv := aesKey[:aes.BlockSize]

	cipherData := make([]byte, len(plainData))
	blockMode := cipher.NewCBCEncrypter(block, iv)
	blockMode.CryptBlocks(cipherData, plainData)

	return cipherData, nil
}

// AesDecrypt 解密
func (e *Encryptor) AesDecrypt(cipherData []byte) ([]byte, error) {
	aesKey := []byte(e.aesKey)
	k := len(aesKey)
	if len(cipherData)%k != 0 {
		return nil, errors.New("crypto/cipher: ciphertext size is not multiple of aes key length")
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	// IV 为 AESKey 的前16字节
	iv := aesKey[:aes.BlockSize]

	blockMode := cipher.NewCBCDecrypter(block, iv)
	plainData := make([]byte, len(cipherData))
	blockMode.CryptBlocks(plainData, cipherData)

	// 去除 PKCS#7 填充
	plainData = PKCS7Unpad(plainData)

	return plainData, nil
}

func (e *Encryptor) ValidAppId(id []byte) bool {
	return string(id) == e.appId
}

func (e *Encryptor) ParseEncryptTextBody(plainText []byte) (*message.Message, error) {
	// Read length
	buf := bytes.NewBuffer(plainText[16:20])
	var length int32
	_ = binary.Read(buf, binary.BigEndian, &length)

	// appId validation
	appIdStart := 20 + length
	id := plainText[appIdStart : int(appIdStart)+len(e.appId)]
	if !e.ValidAppId(id) {
		return nil, errors.New("appid is invalid")
	}

	xmlData := plainText[20 : 20+length]
	messageBody := &message.Message{}
	_ = xml.Unmarshal(xmlData, messageBody)
	messageBody.RawBody = xmlData
	return messageBody, nil
}

// ExtractMsg 从解密后的明文中提取 msg 内容
// 明文格式: random(16字节) + msg_len(4字节) + msg + receiveid
func (e *Encryptor) ExtractMsg(plainText []byte) ([]byte, error) {
	buf := bytes.NewBuffer(plainText[16:20])
	var length int32
	if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	return plainText[20 : 20+length], nil
}
