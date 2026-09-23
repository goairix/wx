package healthcard

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/request"
)

// Call executes a health card request without related WeChat identity fields.
func (client *Client) Call(
	ctx context.Context,
	path string,
	req interface{},
	result interface{},
) error {
	return client.do(ctx, path, req, result, false, "", true)
}

// CallWithRelated executes a request that requires related application identity.
func (client *Client) CallWithRelated(
	ctx context.Context,
	path string,
	req interface{},
	result interface{},
	relateOpenID string,
) error {
	if strings.TrimSpace(client.config.RelatedAppID) == "" {
		return &wxerrors.Error{
			Platform:  "healthcard",
			Operation: path,
			Message:   "related AppID is required",
		}
	}
	if strings.TrimSpace(relateOpenID) == "" {
		return &wxerrors.Error{
			Platform:  "healthcard",
			Operation: path,
			Message:   "related OpenID is required",
		}
	}
	return client.do(ctx, path, req, result, true, relateOpenID, true)
}

func (client *Client) do(
	ctx context.Context,
	path string,
	req interface{},
	result interface{},
	related bool,
	relateOpenID string,
	requireToken bool,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return client.wrapError(path, err)
	}

	appToken := ""
	if requireToken {
		var err error
		appToken, err = client.AppToken(ctx)
		if err != nil {
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return client.wrapError(path, err)
	}

	commonIn := CommonIn{
		AppToken:   appToken,
		RequestID:  client.requestID(),
		HospitalID: client.config.HospitalID,
		Timestamp:  strconv.FormatInt(client.now().Unix(), 10),
		ChannelNum: client.channelNum,
	}
	if related {
		commonIn.RelateAppID = client.config.RelatedAppID
		commonIn.RelateOpenID = relateOpenID
	}

	requestValues, err := structMap(req)
	if err != nil {
		return client.wrapError(path, err)
	}
	commonValues, err := structMap(commonIn)
	if err != nil {
		return client.wrapError(path, err)
	}
	signValues := make(map[string]interface{}, len(requestValues)+len(commonValues))
	for key, value := range commonValues {
		signValues[key] = value
	}
	for key, value := range requestValues {
		signValues[key] = value
	}
	commonIn.Sign = sign(signValues, client.config.AppSecret)

	var envelope responseEnvelope
	meta := new(request.ResponseMeta)
	err = client.transport.Do(ctx, request.Request{
		Operation: path,
		Platform:  "healthcard",
		Method:    http.MethodPost,
		Path:      path,
		Header: http.Header{
			"Content-Type": []string{"application/json;charset=utf-8"},
		},
		Body: requestEnvelope{
			CommonIn: commonIn,
			Req:      req,
		},
		Result: &envelope,
		Meta:   meta,
	})
	if err != nil {
		return client.wrapError(path, err)
	}
	if envelope.CommonOut.ResultCode != 0 {
		return &wxerrors.Error{
			Platform:   "healthcard",
			Operation:  path,
			HTTPStatus: meta.StatusCode,
			Code:       strconv.Itoa(envelope.CommonOut.ResultCode),
			Message:    envelope.CommonOut.ErrMsg,
			RequestID:  envelope.CommonOut.RequestID,
		}
	}
	if result == nil || len(envelope.Rsp) == 0 || string(envelope.Rsp) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Rsp, result); err != nil {
		return client.wrapError(path, fmt.Errorf("decode response: %w", err))
	}
	return nil
}

func (client *Client) wrapError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var platformErr *wxerrors.Error
	if stderrors.As(err, &platformErr) {
		return err
	}
	return &wxerrors.Error{
		Platform:  "healthcard",
		Operation: operation,
		Message:   err.Error(),
		Err:       err,
	}
}

func structMap(value interface{}) (map[string]interface{}, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var values map[string]interface{}
	if err := json.Unmarshal(encoded, &values); err != nil {
		return nil, err
	}
	return values, nil
}
