package openplatform

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	corecache "github.com/goairix/wx/v2/core/cache"
	wxerrors "github.com/goairix/wx/v2/core/errors"
	"github.com/goairix/wx/v2/core/observability"
)

type recordedRequest struct {
	method string
	path   string
	query  string
	body   map[string]interface{}
}

type requestRecorder struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (r *requestRecorder) add(request *http.Request) {
	body := make(map[string]interface{})
	data, _ := io.ReadAll(request.Body)
	if len(data) > 0 {
		_ = json.Unmarshal(data, &body)
	}
	r.mu.Lock()
	r.requests = append(r.requests, recordedRequest{
		method: request.Method,
		path:   request.URL.Path,
		query:  request.URL.RawQuery,
		body:   body,
	})
	r.mu.Unlock()
}

func (r *requestRecorder) all() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedRequest(nil), r.requests...)
}

type trackingCache struct {
	inner *corecache.Memory
	mu    sync.Mutex
	puts  map[string]string
}

func newTrackingCache() *trackingCache {
	return &trackingCache{
		inner: corecache.NewMemory(),
		puts:  make(map[string]string),
	}
}

func (c *trackingCache) Get(
	ctx context.Context,
	key string,
) (string, bool, error) {
	return c.inner.Get(ctx, key)
}

func (c *trackingCache) Put(
	ctx context.Context,
	key string,
	value string,
	ttl time.Duration,
) error {
	c.mu.Lock()
	c.puts[key] = value
	c.mu.Unlock()
	return c.inner.Put(ctx, key, value, ttl)
}

func (c *trackingCache) Delete(ctx context.Context, key string) error {
	return c.inner.Delete(ctx, key)
}

func (c *trackingCache) snapshot() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string]string, len(c.puts))
	for key, value := range c.puts {
		result[key] = value
	}
	return result
}

func TestOpenPlatformRequestContractsAndCredentialIsolation(t *testing.T) {
	recorder := new(requestRecorder)
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		recorder.add(request)
		writer.Header().Set("X-Request-Id", "request-1")
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/cgi-bin/component/api_component_token":
			writeJSON(writer, map[string]interface{}{
				"component_access_token": "component-token",
				"expires_in":             7200,
			})
		case "/cgi-bin/component/api_create_preauthcode":
			writeJSON(writer, map[string]interface{}{
				"pre_auth_code": "pre-auth-code",
				"expires_in":    600,
			})
		case "/cgi-bin/component/api_query_auth":
			writeJSON(writer, map[string]interface{}{
				"authorization_info": map[string]interface{}{
					"authorizer_appid":         "authorizer-app",
					"authorizer_access_token":  "initial-token",
					"authorizer_refresh_token": "refresh-token",
					"expires_in":               7200,
				},
			})
		case "/cgi-bin/component/api_get_authorizer_info":
			writeJSON(writer, map[string]interface{}{
				"authorizer_info": map[string]interface{}{
					"nick_name": "authorized account",
				},
			})
		case "/wxa/gettemplatelist":
			writeJSON(writer, map[string]interface{}{
				"template_list": []map[string]interface{}{
					{"template_id": 12, "user_version": "2.0.0"},
				},
			})
		case "/cgi-bin/component/api_authorizer_token":
			writeJSON(writer, map[string]interface{}{
				"authorizer_access_token":  "authorizer-token",
				"authorizer_refresh_token": "next-refresh-token",
				"expires_in":               7200,
			})
		case "/wxa/release":
			writeJSON(writer, map[string]interface{}{
				"errcode": 0,
				"errmsg":  "ok",
			})
		case "/cgi-bin/user/info":
			writeJSON(writer, map[string]interface{}{
				"openid":   "openid-1",
				"nickname": "reader",
			})
		case "/wxa/business/getuserphonenumber":
			writeJSON(writer, map[string]interface{}{
				"phone_info": map[string]interface{}{
					"phoneNumber": "13800000000",
				},
			})
		case "/cgi-bin/service/get_permanent_code":
			writeJSON(writer, map[string]interface{}{
				"permanent_code": "permanent-code",
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	store := newTrackingCache()
	var hookMu sync.Mutex
	var operations []string
	hook := observability.HookFunc(func(event observability.Event) {
		hookMu.Lock()
		operations = append(operations, event.Operation)
		hookMu.Unlock()
	})
	client, err := NewClient(
		Config{
			AppID:          "component-app",
			AppSecret:      "component-secret",
			Token:          "webhook-token",
			EncodingAESKey: "encoding-key",
		},
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithCache(store),
		WithHook(hook),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.AcceptVerifyTicket(ctx, "component-app", "ticket-1"); err != nil {
		t.Fatal(err)
	}

	credential, err := client.Component().Token(ctx)
	if err != nil || credential.AccessToken != "component-token" {
		t.Fatalf("component credential = %#v, err = %v", credential, err)
	}
	preAuth, err := client.Authorizers().PreAuthorizationCode(ctx)
	if err != nil || preAuth.Code != "pre-auth-code" {
		t.Fatalf("pre-auth code = %#v, err = %v", preAuth, err)
	}
	info, err := client.Authorizers().AuthorizationInfo(ctx, "authorization-code")
	if err != nil || info.AuthorizerAppID != "authorizer-app" {
		t.Fatalf("authorization info = %#v, err = %v", info, err)
	}
	account, err := client.Authorizers().Info(ctx, "authorizer-app")
	if err != nil || account.AuthorizerInfo.Nickname != "authorized account" {
		t.Fatalf("authorizer info = %#v, err = %v", account, err)
	}
	templates, err := client.Templates().List(ctx, -1)
	if err != nil || len(templates) != 1 || templates[0].TemplateID != 12 {
		t.Fatalf("templates = %#v, err = %v", templates, err)
	}
	err = client.Code().ForAuthorizer(
		"authorizer-app",
		"refresh-token",
	).Release(ctx)
	if err != nil {
		t.Fatal(err)
	}

	officialClient, err := client.AuthorizedOfficial(
		"official-app",
		"official-refresh-token",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := officialClient.Users().Info(ctx, "openid-1"); err != nil {
		t.Fatal(err)
	}
	miniappClient, err := client.AuthorizedMiniApp(
		"miniapp-app",
		"miniapp-refresh-token",
	)
	if err != nil {
		t.Fatal(err)
	}
	phone, err := miniappClient.Users().GetPhoneNumber(ctx, "code-1", "openid-1")
	if err != nil || phone.PhoneNumber != "13800000000" {
		t.Fatalf("phone = %#v, err = %v", phone, err)
	}
	permanentCode, err := client.WorkAuthorizer().PermanentCode(
		ctx,
		"suite-token",
		"temporary-code",
	)
	if err != nil || permanentCode.PermanentCode != "permanent-code" {
		t.Fatalf("permanent code = %#v, err = %v", permanentCode, err)
	}

	assertRequestContracts(t, recorder.all())
	assertCredentialKeys(t, store.snapshot())
	hookMu.Lock()
	defer hookMu.Unlock()
	if !containsOperation(operations, "openplatform.code.release") {
		t.Fatalf("release operation was not observed: %v", operations)
	}
}

func TestOpenPlatformErrorIncludesTransportMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		writer.Header().Set("X-Request-Id", "request-error")
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/cgi-bin/component/api_component_token" {
			writeJSON(writer, map[string]interface{}{
				"component_access_token": "component-token",
				"expires_in":             7200,
			})
			return
		}
		writer.WriteHeader(http.StatusTooManyRequests)
		writeJSON(writer, map[string]interface{}{
			"errcode": 45009,
			"errmsg":  "rate limit",
		})
	}))
	defer server.Close()

	client, err := NewClient(
		Config{AppID: "component-app", AppSecret: "component-secret"},
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Component().SetVerifyTicket(
		context.Background(),
		"ticket-1",
	); err != nil {
		t.Fatal(err)
	}
	_, err = client.Templates().Drafts(context.Background())
	var structured *wxerrors.Error
	if !stderrors.As(err, &structured) {
		t.Fatalf("expected structured error, got %T: %v", err, err)
	}
	if structured.HTTPStatus != http.StatusTooManyRequests ||
		structured.RequestID != "request-error" ||
		structured.Operation != "openplatform.template.drafts" {
		t.Fatalf("unexpected structured error: %#v", structured)
	}
}

func TestAuthorizedFactoriesInheritTransportCacheAndHook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/cgi-bin/component/api_component_token":
			writeJSON(writer, map[string]interface{}{
				"component_access_token": "component-token",
				"expires_in":             7200,
			})
		case "/cgi-bin/component/api_authorizer_token":
			var body struct {
				AppID string `json:"authorizer_appid"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode authorizer token request: %v", err)
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			if request.URL.Query().Get("component_access_token") != "component-token" {
				t.Errorf("unexpected component token query: %s", request.URL.RawQuery)
			}
			writeJSON(writer, map[string]interface{}{
				"authorizer_access_token": "token-" + body.AppID,
				"expires_in":              7200,
			})
		case "/cgi-bin/user/info":
			if request.URL.Query().Get("access_token") != "token-official-app" {
				t.Errorf("official request query: %s", request.URL.RawQuery)
			}
			writeJSON(writer, map[string]interface{}{
				"openid": "openid-1",
			})
		case "/wxa/business/getuserphonenumber":
			if request.URL.Query().Get("access_token") != "token-miniapp-app" {
				t.Errorf("miniapp request query: %s", request.URL.RawQuery)
			}
			writeJSON(writer, map[string]interface{}{
				"phone_info": map[string]interface{}{
					"phoneNumber": "13800000000",
				},
			})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	store := newTrackingCache()
	var hookMu sync.Mutex
	var operations []string
	hook := observability.HookFunc(func(event observability.Event) {
		hookMu.Lock()
		operations = append(operations, event.Operation)
		hookMu.Unlock()
	})
	client, err := NewClient(
		Config{
			AppID:     "component-app",
			AppSecret: "component-secret",
		},
		WithHTTPClient(server.Client()),
		WithBaseURL(server.URL),
		WithCache(store),
		WithHook(hook),
	)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := client.Component().SetVerifyTicket(ctx, "ticket-1"); err != nil {
		t.Fatal(err)
	}

	officialClient, err := client.AuthorizedOfficial(
		"official-app",
		"official-refresh-secret",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := officialClient.Users().Info(ctx, "openid-1"); err != nil {
		t.Fatal(err)
	}

	miniappClient, err := client.AuthorizedMiniApp(
		"miniapp-app",
		"miniapp-refresh-secret",
	)
	if err != nil {
		t.Fatal(err)
	}
	phone, err := miniappClient.Users().GetPhoneNumber(
		ctx,
		"phone-code",
		"openid-1",
	)
	if err != nil || phone.PhoneNumber != "13800000000" {
		t.Fatalf("phone = %#v, err = %v", phone, err)
	}

	hookMu.Lock()
	recordedOperations := append([]string(nil), operations...)
	hookMu.Unlock()
	if !containsOperation(recordedOperations, "official.user.info") {
		t.Fatalf("official operation was not observed: %v", recordedOperations)
	}
	if !containsOperation(recordedOperations, "miniapp.user.phone") {
		t.Fatalf("miniapp operation was not observed: %v", recordedOperations)
	}

	assertFactoryCredentialKeys(t, store.snapshot())
}

func TestAcceptVerifyTicketFailsClosed(t *testing.T) {
	client, err := NewClient(Config{
		AppID:     "component-app",
		AppSecret: "component-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = client.AcceptVerifyTicket(
		context.Background(),
		"another-component",
		"ticket-1",
	)
	if err == nil {
		t.Fatal("expected mismatched component AppID to be rejected")
	}
	_, ok, getErr := client.Component().VerifyTicket(context.Background())
	if getErr != nil || ok {
		t.Fatalf("ticket should not be stored: ok=%v err=%v", ok, getErr)
	}
}

func assertRequestContracts(t *testing.T, requests []recordedRequest) {
	t.Helper()
	byPath := make(map[string]recordedRequest, len(requests))
	for _, request := range requests {
		byPath[request.path] = request
	}

	componentToken := byPath["/cgi-bin/component/api_component_token"]
	if componentToken.method != http.MethodPost ||
		componentToken.body["component_verify_ticket"] != "ticket-1" {
		t.Fatalf("component token request = %#v", componentToken)
	}
	preAuth := byPath["/cgi-bin/component/api_create_preauthcode"]
	if preAuth.method != http.MethodPost ||
		!strings.Contains(preAuth.query, "component_access_token=component-token") {
		t.Fatalf("pre-auth request = %#v", preAuth)
	}
	queryAuth := byPath["/cgi-bin/component/api_query_auth"]
	if queryAuth.body["authorization_code"] != "authorization-code" {
		t.Fatalf("authorization request = %#v", queryAuth)
	}
	authorizerInfo := byPath["/cgi-bin/component/api_get_authorizer_info"]
	if authorizerInfo.body["authorizer_appid"] != "authorizer-app" {
		t.Fatalf("authorizer info request = %#v", authorizerInfo)
	}
	templateList := byPath["/wxa/gettemplatelist"]
	if templateList.method != http.MethodGet ||
		!strings.Contains(templateList.query, "access_token=component-token") {
		t.Fatalf("template list request = %#v", templateList)
	}
	release := byPath["/wxa/release"]
	if release.method != http.MethodPost ||
		!strings.Contains(release.query, "access_token=authorizer-token") {
		t.Fatalf("release request = %#v", release)
	}
	authorizerToken := byPath["/cgi-bin/component/api_authorizer_token"]
	if authorizerToken.method != http.MethodPost ||
		!strings.Contains(
			authorizerToken.query,
			"component_access_token=component-token",
		) ||
		authorizerToken.body["authorizer_appid"] != "miniapp-app" ||
		authorizerToken.body["authorizer_refresh_token"] != "miniapp-refresh-token" {
		t.Fatalf("authorizer token request = %#v", authorizerToken)
	}
	officialUser := byPath["/cgi-bin/user/info"]
	if !strings.Contains(
		officialUser.query,
		"access_token=authorizer-token",
	) {
		t.Fatalf("authorized official request = %#v", officialUser)
	}
	miniappPhone := byPath["/wxa/business/getuserphonenumber"]
	if !strings.Contains(
		miniappPhone.query,
		"access_token=authorizer-token",
	) {
		t.Fatalf("authorized miniapp request = %#v", miniappPhone)
	}
	work := byPath["/cgi-bin/service/get_permanent_code"]
	if !strings.Contains(work.query, "suite_access_token=suite-token") ||
		work.body["auth_code"] != "temporary-code" {
		t.Fatalf("work authorizer request = %#v", work)
	}
}

func assertCredentialKeys(t *testing.T, entries map[string]string) {
	t.Helper()
	var componentKey string
	var authorizerKey string
	for key, value := range entries {
		if strings.Contains(key, "component-secret") ||
			strings.Contains(key, "refresh-token") {
			t.Fatalf("credential cache key contains secret material: %q", key)
		}
		if strings.Contains(value, "component-token") {
			componentKey = key
		}
		if strings.Contains(value, "authorizer-token") {
			authorizerKey = key
		}
	}
	if componentKey == "" || authorizerKey == "" {
		t.Fatalf("missing credential cache entries: %#v", entries)
	}
	if componentKey == authorizerKey {
		t.Fatalf("component and authorizer credentials share key %q", componentKey)
	}
}

func assertFactoryCredentialKeys(t *testing.T, entries map[string]string) {
	t.Helper()
	var officialKey string
	var miniappKey string
	for key, value := range entries {
		if strings.Contains(key, "official-refresh-secret") ||
			strings.Contains(key, "miniapp-refresh-secret") {
			t.Fatalf("factory credential cache key contains refresh token: %q", key)
		}
		if strings.Contains(value, "token-official-app") {
			officialKey = key
		}
		if strings.Contains(value, "token-miniapp-app") {
			miniappKey = key
		}
	}
	if officialKey == "" || miniappKey == "" {
		t.Fatalf("missing factory credential cache entries: %#v", entries)
	}
	if officialKey == miniappKey {
		t.Fatalf("authorized factories share credential key %q", officialKey)
	}
}

func containsOperation(operations []string, target string) bool {
	for _, operation := range operations {
		if operation == target {
			return true
		}
	}
	return false
}

func writeJSON(writer http.ResponseWriter, value interface{}) {
	_ = json.NewEncoder(writer).Encode(value)
}
