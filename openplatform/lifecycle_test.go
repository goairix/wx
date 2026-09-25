package openplatform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type lifecycleAPI interface {
	UpdateAuthorizer(context.Context, string, string) error
	RevokeAuthorizer(context.Context, string) error
}

func requireLifecycle(t *testing.T, client *Client) lifecycleAPI {
	t.Helper()
	api, ok := any(client).(lifecycleAPI)
	if !ok {
		t.Fatal("client does not implement explicit authorizer lifecycle")
	}
	return api
}

type hostTransport func(*http.Request) (*http.Response, error)

func (f hostTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWorkAuthorizerUsesEnterpriseHost(t *testing.T) {
	var host string
	client, err := NewClient(Config{AppID: "component", AppSecret: "secret"}, WithHTTPClient(&http.Client{Transport: hostTransport(func(r *http.Request) (*http.Response, error) {
		host = r.URL.Host
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"permanent_code":"code"}`))}, nil
	})}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.WorkAuthorizer().PermanentCode(context.Background(), "suite", "code"); err != nil {
		t.Fatal(err)
	}
	if host != "qyapi.weixin.qq.com" {
		t.Fatalf("enterprise host = %q", host)
	}
}

func lifecycleServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var used []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/component/api_component_token":
			writeJSON(w, map[string]any{"component_access_token": "component-token", "expires_in": 7200})
		case "/cgi-bin/component/api_authorizer_token":
			var body struct {
				RefreshToken string `json:"authorizer_refresh_token"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				return
			}
			mu.Lock()
			used = append(used, body.RefreshToken)
			mu.Unlock()
			writeJSON(w, map[string]any{"authorizer_access_token": "access-" + body.RefreshToken, "authorizer_refresh_token": body.RefreshToken + "-rotated", "expires_in": 7200})
		case "/wxa/release", "/cgi-bin/user/info":
			writeJSON(w, map[string]any{"errcode": 0, "openid": "reader"})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server, func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), used...) }
}

func lifecycleClient(t *testing.T, server *httptest.Server, options ...Option) *Client {
	t.Helper()
	opts := append([]Option{WithHTTPClient(server.Client()), WithBaseURL(server.URL)}, options...)
	client, err := NewClient(Config{AppID: "component", AppSecret: "secret"}, opts...)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Component().SetVerifyTicket(context.Background(), "ticket"); err != nil {
		t.Fatal(err)
	}
	return client
}

func TestAuthorizerLifecycleUpdatesExistingChildren(t *testing.T) {
	server, used := lifecycleServer(t)
	client := lifecycleClient(t, server)
	lifecycle := requireLifecycle(t, client)
	ctx := context.Background()
	child, err := client.AuthorizedOfficial("authorizer", "initial")
	if err != nil {
		t.Fatal(err)
	}
	code := client.Code().ForAuthorizer("authorizer", "initial")
	if _, err := child.Users().Info(ctx, "reader"); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.UpdateAuthorizer(ctx, "authorizer", "reauthorized"); err != nil {
		t.Fatal(err)
	}
	client.Code().ForAuthorizer("authorizer", "stale-factory-value")
	if err := code.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := child.Users().Info(ctx, "reader"); err != nil {
		t.Fatal(err)
	}
	if got := used(); !reflect.DeepEqual(got, []string{"initial", "reauthorized"}) {
		t.Fatalf("refresh tokens = %v", got)
	}
	if err := lifecycle.RevokeAuthorizer(ctx, "authorizer"); err != nil {
		t.Fatal(err)
	}
	if err := code.Release(ctx); err == nil {
		t.Fatal("existing child used revoked authorization")
	}
	if _, err := child.Users().Info(ctx, "reader"); err == nil {
		t.Fatal("existing official client used revoked authorization")
	}
	if got := used(); len(got) != 2 {
		t.Fatalf("revocation made refresh request: %v", got)
	}
	if err := lifecycle.UpdateAuthorizer(ctx, "authorizer", "restored"); err != nil {
		t.Fatal(err)
	}
	if err := code.Release(ctx); err != nil {
		t.Fatal(err)
	}
}

type memoryRefreshRepository struct {
	mu                          sync.Mutex
	token                       string
	saveErr, loadErr, deleteErr error
	calls                       int
}

func (r *memoryRefreshRepository) SaveRefreshToken(_ context.Context, _ string, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.saveErr != nil {
		return r.saveErr
	}
	r.token = token
	return nil
}
func (r *memoryRefreshRepository) LoadRefreshToken(context.Context, string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	return r.token, r.loadErr
}
func (r *memoryRefreshRepository) DeleteRefreshToken(context.Context, string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.token = ""
	return nil
}

// Keep missing lifecycle APIs as runtime assertions so the first RED run proves
// the existing behavior before the new repository option is introduced.
func TestAuthorizerLifecyclePersistenceFailure(t *testing.T) {
	server, used := lifecycleServer(t)
	fail := false
	storeErr := errors.New("store unavailable")
	client := lifecycleClient(t, server, WithRefreshTokenStore(RefreshTokenStoreFunc(func(context.Context, string, string) error {
		if fail {
			return storeErr
		}
		return nil
	})))
	api := requireLifecycle(t, client)
	ctx := context.Background()
	child := client.Code().ForAuthorizer("authorizer", "initial")
	if err := child.Release(ctx); err != nil {
		t.Fatal(err)
	}
	fail = true
	if err := api.UpdateAuthorizer(ctx, "authorizer", "replacement"); !errors.Is(err, storeErr) {
		t.Fatalf("update error = %v", err)
	}
	if err := child.Release(ctx); !errors.Is(err, storeErr) {
		t.Fatalf("pending persistence error = %v", err)
	}
	if got := used(); len(got) != 1 {
		t.Fatalf("refresh before persistence succeeded: %v", got)
	}
	fail = false
	if err := child.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if got := used(); fmt.Sprint(got) != "[initial replacement]" {
		t.Fatalf("refresh tokens = %v", got)
	}
}

func TestRefreshTokenRepositoryCoordinatesRoots(t *testing.T) {
	server, used := lifecycleServer(t)
	repository := &memoryRefreshRepository{}
	first := lifecycleClient(t, server, WithRefreshTokenRepository(repository))
	second := lifecycleClient(t, server, WithRefreshTokenRepository(repository))
	ctx := context.Background()
	firstChild := first.Code().ForAuthorizer("authorizer", "stale")
	secondChild := second.Code().ForAuthorizer("authorizer", "also-stale")
	if repository.calls != 0 {
		t.Fatal("factory accessed refresh repository")
	}
	if err := firstChild.Release(ctx); err == nil {
		t.Fatal("empty repository accepted factory token")
	}
	if err := first.UpdateAuthorizer(ctx, "authorizer", "initial"); err != nil {
		t.Fatal(err)
	}
	if err := firstChild.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if err := secondChild.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if err := first.authorizerManager("authorizer", "").Invalidate(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if err := firstChild.Release(ctx); err != nil {
		t.Fatal(err)
	}
	want := []string{"initial", "initial-rotated", "initial-rotated-rotated"}
	if got := used(); !reflect.DeepEqual(got, want) {
		t.Fatalf("refresh tokens = %v, want %v", got, want)
	}
	if err := second.RevokeAuthorizer(ctx, "authorizer"); err != nil {
		t.Fatal(err)
	}
	if err := first.authorizerManager("authorizer", "").Invalidate(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if err := firstChild.Release(ctx); err == nil {
		t.Fatal("root ignored repository revocation")
	}
}

func TestRepositoryErrorsFailClosedAndCanRetry(t *testing.T) {
	server, used := lifecycleServer(t)
	repo := &memoryRefreshRepository{}
	client := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
	ctx := context.Background()
	child := client.Code().ForAuthorizer("authorizer", "initial")
	if err := client.UpdateAuthorizer(ctx, "authorizer", "initial"); err != nil {
		t.Fatal(err)
	}
	repo.loadErr = errors.New("load failed")
	if err := child.Release(ctx); !errors.Is(err, repo.loadErr) {
		t.Fatalf("load error = %v", err)
	}
	if len(used()) != 0 {
		t.Fatal("load failure triggered refresh")
	}
	repo.loadErr = nil
	if err := child.Release(ctx); err != nil {
		t.Fatal(err)
	}
	repo.deleteErr = errors.New("delete failed")
	if err := client.RevokeAuthorizer(ctx, "authorizer"); !errors.Is(err, repo.deleteErr) {
		t.Fatalf("delete error = %v", err)
	}
	if err := child.Release(ctx); err == nil {
		t.Fatal("failed deletion restored local revoked authorization")
	}
	if len(used()) != 1 {
		t.Fatal("failed deletion triggered refresh")
	}
	repo.deleteErr = nil
	if err := client.RevokeAuthorizer(ctx, "authorizer"); err != nil {
		t.Fatal(err)
	}
	if repo.token != "" {
		t.Fatal("retry did not delete refresh token")
	}
	if err := child.Release(ctx); err == nil {
		t.Fatal("revoked authorization accepted")
	}
}

func TestRepositoryRetriesRotatedTokenBeforeRefreshing(t *testing.T) {
	server, used := lifecycleServer(t)
	repo := &memoryRefreshRepository{token: "initial", saveErr: errors.New("save failed")}
	client := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
	child := client.Code().ForAuthorizer("authorizer", "stale")
	ctx := context.Background()
	if err := child.Release(ctx); !errors.Is(err, repo.saveErr) {
		t.Fatalf("save error = %v", err)
	}
	if err := child.Release(ctx); !errors.Is(err, repo.saveErr) {
		t.Fatalf("pending save error = %v", err)
	}
	if got := used(); !reflect.DeepEqual(got, []string{"initial"}) {
		t.Fatalf("refreshed before persistence: %v", got)
	}
	repo.saveErr = nil
	if err := child.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if got := used(); !reflect.DeepEqual(got, []string{"initial", "initial-rotated"}) {
		t.Fatalf("refresh tokens = %v", got)
	}
}

func TestLifecycleDiscardsPendingRotation(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		t.Run(fmt.Sprintf("revoke=%v", revoke), func(t *testing.T) {
			server, used := lifecycleServer(t)
			repo := &memoryRefreshRepository{token: "initial", saveErr: errors.New("save failed")}
			client := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
			child := client.Code().ForAuthorizer("authorizer", "stale")
			ctx := context.Background()
			if err := child.Release(ctx); err == nil {
				t.Fatal("expected pending rotated token")
			}
			repo.saveErr = nil
			if revoke {
				if err := client.RevokeAuthorizer(ctx, "authorizer"); err != nil {
					t.Fatal(err)
				}
				if err := child.Release(ctx); err == nil {
					t.Fatal("pending rotation restored revoked authorization")
				}
				if repo.token != "" {
					t.Fatalf("repository token = %q", repo.token)
				}
				if len(used()) != 1 {
					t.Fatal("revoked authorization refreshed")
				}
			} else {
				if err := client.UpdateAuthorizer(ctx, "authorizer", "replacement"); err != nil {
					t.Fatal(err)
				}
				if err := child.Release(ctx); err != nil {
					t.Fatal(err)
				}
				if got := used(); !reflect.DeepEqual(got, []string{"initial", "replacement"}) {
					t.Fatalf("refresh tokens = %v", got)
				}
			}
		})
	}
}

func TestPendingRotationDoesNotRestoreRepositoryRevocation(t *testing.T) {
	server, used := lifecycleServer(t)
	repo := &memoryRefreshRepository{token: "initial", saveErr: errors.New("save failed")}
	first := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
	second := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
	ctx := context.Background()
	child := first.Code().ForAuthorizer("authorizer", "stale")
	if err := child.Release(ctx); !errors.Is(err, repo.saveErr) {
		t.Fatalf("initial refresh error = %v", err)
	}
	if err := second.RevokeAuthorizer(ctx, "authorizer"); err != nil {
		t.Fatal(err)
	}
	repo.saveErr = nil
	if err := child.Release(ctx); err == nil {
		t.Fatal("pending rotation restored a revoked authorization")
	}
	if repo.token != "" {
		t.Fatalf("repository token = %q after revocation", repo.token)
	}
	if got := used(); !reflect.DeepEqual(got, []string{"initial"}) {
		t.Fatalf("refresh tokens = %v", got)
	}
}

func TestPendingDeletionDoesNotRemoveRepositoryReauthorization(t *testing.T) {
	server, used := lifecycleServer(t)
	repo := &memoryRefreshRepository{token: "initial"}
	first := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
	second := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
	ctx := context.Background()
	child := first.Code().ForAuthorizer("authorizer", "stale")
	if err := child.Release(ctx); err != nil {
		t.Fatal(err)
	}
	repo.deleteErr = errors.New("delete failed")
	if err := first.RevokeAuthorizer(ctx, "authorizer"); !errors.Is(err, repo.deleteErr) {
		t.Fatalf("revoke error = %v", err)
	}
	if err := second.UpdateAuthorizer(ctx, "authorizer", "replacement"); err != nil {
		t.Fatal(err)
	}
	repo.deleteErr = nil
	if err := child.Release(ctx); err != nil {
		t.Fatalf("pending deletion removed reauthorization: %v", err)
	}
	if repo.token != "replacement-rotated" {
		t.Fatalf("repository token = %q", repo.token)
	}
	if got := used(); !reflect.DeepEqual(got, []string{"initial", "replacement"}) {
		t.Fatalf("refresh tokens = %v", got)
	}
}

type uncertainRefreshRepository struct {
	memoryRefreshRepository
	failNextSave bool
	saves        int
}

func (r *uncertainRefreshRepository) SaveRefreshToken(ctx context.Context, appID, token string) error {
	r.saves++
	if err := r.memoryRefreshRepository.SaveRefreshToken(ctx, appID, token); err != nil {
		return err
	}
	if r.failNextSave {
		r.failNextSave = false
		return errors.New("save acknowledgement lost")
	}
	return nil
}
func TestPendingRotationAlreadyPersistedIsNotSavedAgain(t *testing.T) {
	server, used := lifecycleServer(t)
	repo := &uncertainRefreshRepository{memoryRefreshRepository: memoryRefreshRepository{token: "initial"}, failNextSave: true}
	client := lifecycleClient(t, server, WithRefreshTokenRepository(repo))
	child := client.Code().ForAuthorizer("authorizer", "stale")
	ctx := context.Background()
	if err := child.Release(ctx); err == nil {
		t.Fatal("expected uncertain save result")
	}
	if err := child.Release(ctx); err != nil {
		t.Fatal(err)
	}
	if repo.saves != 2 {
		t.Fatalf("save calls = %d, want one per rotation", repo.saves)
	}
	if got := used(); !reflect.DeepEqual(got, []string{"initial", "initial-rotated"}) {
		t.Fatalf("refresh tokens = %v", got)
	}
}
