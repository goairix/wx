package work_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/goairix/wx/v2/work"
)

func ExampleClient_Agent() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/cgi-bin/gettoken":
			_, _ = io.WriteString(w, `{"access_token":"token","expires_in":7200}`)
		case "/cgi-bin/agent/get":
			_, _ = io.WriteString(w, `{"errcode":0,"agentid":1000005,"name":"HR助手"}`)
		}
	}))
	defer server.Close()

	client, err := work.NewClient(work.Config{CorpID: "corp", CorpSecret: "secret"}, work.WithBaseURL(server.URL))
	if err != nil {
		panic(err)
	}
	detail, err := client.Agent().Get(context.Background(), 1000005)
	if err != nil {
		panic(err)
	}
	fmt.Println(detail.AgentID, detail.Name)
	// Output: 1000005 HR助手
}
