package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goairix/wx/v2/core/cache"
)

type updateContextKey struct{}

func TestUpdateCoordinatesWithRefreshAndPreservesContext(t *testing.T) {
	store := cache.NewMemory()
	entered, resume := make(chan struct{}), make(chan struct{})
	m := NewManager("mobileapp.oauth", "user", store, ProviderFunc(func(context.Context) (Credential, error) {
		close(entered)
		<-resume
		return Credential{AccessToken: "refreshed", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	updater := NewManager("mobileapp.oauth", "user", store, nil)
	refreshDone := make(chan error, 1)
	go func() { _, err := m.Token(context.Background()); refreshDone <- err }()
	<-entered
	ctx := context.WithValue(context.Background(), updateContextKey{}, "replacement")
	updateDone := make(chan error, 1)
	go func() {
		_, err := updater.Update(ctx, ProviderFunc(func(got context.Context) (Credential, error) {
			if got != ctx {
				return Credential{}, errors.New("update replaced original context")
			}
			return Credential{AccessToken: "issued", ExpiresAt: time.Now().Add(time.Hour)}, nil
		}))
		updateDone <- err
	}()
	waitForParticipants(t, store, m.cacheKey, m.coordinator, 2)
	close(resume)
	if err := <-refreshDone; err != nil {
		t.Fatal(err)
	}
	if err := <-updateDone; err != nil {
		t.Fatal(err)
	}
	credential, err := m.Token(ctx)
	if err != nil || credential.AccessToken != "issued" {
		t.Fatalf("credential=%v err=%v", credential, err)
	}
}

func TestUpdateWaitersRecheckCacheAfterPersistenceFailure(t *testing.T) {
	store := cache.NewMemory()
	entered, resume := make(chan struct{}), make(chan struct{})
	m := NewManager("mobileapp.oauth", "failure", store, ProviderFunc(func(context.Context) (Credential, error) {
		return Credential{AccessToken: "recovered", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	updater := m
	failure := errors.New("persist failed")
	updateDone := make(chan error, 1)
	go func() {
		_, err := updater.Update(context.Background(), ProviderFunc(func(context.Context) (Credential, error) {
			close(entered)
			<-resume
			return Credential{}, failure
		}))
		updateDone <- err
	}()
	<-entered
	waiterDone := make(chan error, 1)
	go func() {
		got, err := m.Token(context.Background())
		if err == nil && got.AccessToken != "recovered" {
			err = errors.New("waiter received empty update credential")
		}
		waiterDone <- err
	}()
	waitForParticipants(t, store, m.cacheKey, m.coordinator, 2)
	close(resume)
	if err := <-updateDone; !errors.Is(err, failure) {
		t.Fatalf("update error=%v", err)
	}
	if err := <-waiterDone; err != nil {
		t.Fatal(err)
	}
}

func TestUpdateWaitingContextCanCancel(t *testing.T) {
	store := cache.NewMemory()
	entered, resume := make(chan struct{}), make(chan struct{})
	m := NewManager("mobileapp.oauth", "cancel", store, ProviderFunc(func(context.Context) (Credential, error) {
		close(entered)
		<-resume
		return Credential{AccessToken: "existing", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}))
	updater := m
	done := make(chan error, 1)
	go func() { _, err := m.Token(context.Background()); done <- err }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	waiting := make(chan error, 1)
	go func() {
		_, err := updater.Update(ctx, ProviderFunc(func(context.Context) (Credential, error) {
			return Credential{}, errors.New("canceled update called provider")
		}))
		waiting <- err
	}()
	waitForParticipants(t, store, m.cacheKey, m.coordinator, 2)
	cancel()
	err := <-waiting
	close(resume)
	if err != context.Canceled {
		t.Fatalf("update error=%v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
