package observability

import (
	"errors"
	"testing"
	"time"
)

func TestHookReceivesOperationAndRequestID(t *testing.T) {
	var got Event
	hook := HookFunc(func(event Event) { got = event })
	hook.OnResponse(Event{Operation: "work.contact.user.get", StatusCode: 200, RequestID: "req-3"})
	if got.Operation != "work.contact.user.get" || got.RequestID != "req-3" {
		t.Fatalf("unexpected event: %+v", got)
	}
}

func TestHookFuncForwardsRequestAndResponseEvents(t *testing.T) {
	var events []Event
	hook := HookFunc(func(event Event) { events = append(events, event) })
	requestEvent := Event{Operation: "test.request", Platform: "official"}
	responseEvent := Event{Operation: "test.response", StatusCode: 200, RequestID: "req-1", Duration: time.Second, Err: errors.New("boom")}

	hook.OnRequest(requestEvent)
	hook.OnResponse(responseEvent)

	if len(events) != 2 || events[0].Operation != requestEvent.Operation || events[1].RequestID != responseEvent.RequestID {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestNilHookFuncIsSafe(t *testing.T) {
	var hook HookFunc
	hook.OnRequest(Event{Operation: "test.request"})
	hook.OnResponse(Event{Operation: "test.response"})
}
