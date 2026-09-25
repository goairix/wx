package observability

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHookReceivesOperationAndRequestID(t *testing.T) {
	var got Event
	ctx := context.Background()
	hook := HookFunc(func(event Event) { got = event })
	hook.OnResponse(Event{Context: ctx, Operation: "work.contact.user.get", StatusCode: 200, RequestID: "req-3"})
	if got.Operation != "work.contact.user.get" || got.RequestID != "req-3" {
		t.Fatalf("unexpected event: %+v", got)
	}
	if got.Context != ctx {
		t.Fatal("hook did not preserve event context")
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

func TestObserverFuncForwardsContextEventAndFinish(t *testing.T) {
	ctx := context.Background()
	event := Event{Operation: "test.observe", Attempt: 2, MaxAttempts: 3}
	finished := false
	observer := ObserverFunc(func(got context.Context, gotEvent Event) (context.Context, func(Event)) {
		if got != ctx || gotEvent.Operation != event.Operation || gotEvent.Attempt != 2 {
			t.Fatal("observer lost context or event")
		}
		return got, func(end Event) { finished = end.StatusCode == 200 }
	})
	got, finish := observer.Start(ctx, event)
	if got != ctx || finish == nil {
		t.Fatal("observer lost return values")
	}
	finish(Event{StatusCode: 200})
	if !finished {
		t.Fatal("finish not forwarded")
	}
	var nilObserver ObserverFunc
	got, finish = nilObserver.Start(ctx, event)
	if got != ctx || finish != nil {
		t.Fatal("nil observer did not preserve context")
	}
}
