package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// MaxBodyBytes is the maximum accepted webhook request body size.
const MaxBodyBytes = 1 << 20

// ErrBodyTooLarge is returned before dispatching an oversized webhook.
var ErrBodyTooLarge = errors.New("webhook: request body too large")

// Payload is the parsed webhook body. Raw always contains the original body.
type Payload struct {
	Raw    []byte
	Format string
	Values map[string]string
}

// Handler processes a webhook payload.
type Handler interface {
	Handle(context.Context, Payload) (Response, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(context.Context, Payload) (Response, error)

func (f HandlerFunc) Handle(ctx context.Context, payload Payload) (Response, error) {
	return f(ctx, payload)
}

// Response is an HTTP response returned by a webhook handler.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// EmptyResponse returns a successful empty response.
func EmptyResponse() Response {
	return Response{Status: http.StatusOK}
}

type ErrorResponse func(error) Response
type Option func(*handler)

// WithErrorResponse supplies the platform-specific error response strategy.
func WithErrorResponse(policy ErrorResponse) Option {
	return func(h *handler) { h.errorResponse = policy }
}

type handler struct {
	next          Handler
	errorResponse ErrorResponse
}

// NewHandler builds an HTTP handler for XML and JSON webhook bodies.
func NewHandler(next Handler, options ...Option) http.Handler {
	return newHandler(next, options...)
}

func newHandler(next Handler, options ...Option) *handler {
	h := &handler{
		next: next,
		errorResponse: func(error) Response {
			return Response{Status: http.StatusBadRequest}
		},
	}
	for _, option := range options {
		if option != nil {
			option(h)
		}
	}
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := readPayload(r)
	if err != nil {
		h.write(w, h.errorResponse(err))
		return
	}
	response, err := h.next.Handle(r.Context(), payload)
	if err != nil {
		response = h.errorResponse(err)
	}
	h.write(w, response)
}

func (h *handler) write(w http.ResponseWriter, response Response) {
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	if len(response.Body) > 0 {
		_, _ = w.Write(response.Body)
	}
}

func readPayload(r *http.Request) (Payload, error) {
	body, err := readBody(r.Body)
	if err != nil {
		return Payload{}, err
	}
	payload := Payload{Raw: body, Values: make(map[string]string)}
	format := r.Header.Get("Content-Type")
	if len(bytes.TrimSpace(body)) > 0 && bytes.TrimSpace(body)[0] == '{' || contains(format, "json") {
		payload.Format = "json"
		var value interface{}
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return Payload{}, fmt.Errorf("parse webhook json: %w", err)
		}
		var trailing interface{}
		if err := decoder.Decode(&trailing); err != io.EOF {
			return Payload{}, fmt.Errorf("parse webhook json: trailing content")
		}
		flattenJSON(payload.Values, "", value)
		return payload, nil
	}
	payload.Format = "xml"
	var node xmlNode
	if err := xml.Unmarshal(body, &node); err != nil {
		return Payload{}, fmt.Errorf("parse webhook xml: %w", err)
	}
	flattenXML(payload.Values, node)
	return payload, nil
}

func readBody(reader io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, int64(MaxBodyBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > MaxBodyBytes {
		return nil, ErrBodyTooLarge
	}
	return body, nil
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

type xmlNode struct {
	XMLName  xml.Name
	Text     string    `xml:",chardata"`
	Children []xmlNode `xml:",any"`
}

func flattenXML(values map[string]string, node xmlNode) {
	if len(node.Children) == 0 {
		values[node.XMLName.Local] = node.Text
		return
	}
	for _, child := range node.Children {
		flattenXML(values, child)
	}
}
func flattenJSON(values map[string]string, prefix string, value interface{}) {
	switch current := value.(type) {
	case map[string]interface{}:
		for key, item := range current {
			name := key
			if prefix != "" {
				name = prefix + "." + key
			}
			flattenJSON(values, name, item)
		}
	case []interface{}:
		for i, item := range current {
			flattenJSON(values, fmt.Sprintf("%s.%d", prefix, i), item)
		}
	case nil:
		values[prefix] = ""
	default:
		values[prefix] = fmt.Sprint(current)
	}
}
