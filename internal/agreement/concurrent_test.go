package agreement

import (
	"context"
	"errors"
	"testing"

	"skillswap/domain"
)

func TestConcurrentAcceptOnSharedStore(t *testing.T) {
	store := scenarioStore(t)
	services := [2]*Service{NewService(store), NewService(store)}
	const attempts = 32
	type result struct {
		response domain.Response
		changed  bool
		err      error
	}
	results := make(chan result, attempts)
	start := make(chan struct{})
	for i := range attempts {
		go func(service *Service) {
			<-start
			resp, changed, err := service.AcceptContext(t.Context(), "author", "response-1")
			results <- result{resp, changed, err}
		}(services[i%len(services)])
	}
	close(start)

	changes := 0
	for range attempts {
		got := <-results
		if got.err != nil || got.response.Status != domain.ResponseAccepted {
			t.Errorf("accept: %+v", got)
		}
		if got.changed {
			changes++
		}
	}
	if changes != 1 {
		t.Fatalf("committed changes: got %d, want 1", changes)
	}
	responses, err := store.ListResponses()
	if err != nil || len(responses) != 1 || responses[0].ID != "response-1" || responses[0].Status != domain.ResponseAccepted {
		t.Fatalf("stored: %+v, %v", responses, err)
	}
}

func TestAcceptCanceledLeavesResponsePending(t *testing.T) {
	store := scenarioStore(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, changed, err := NewService(store).AcceptContext(ctx, "author", "response-1")
	if !errors.Is(err, context.Canceled) || changed || got != (domain.Response{}) {
		t.Fatalf("canceled accept: %+v, %t, %v", got, changed, err)
	}
	stored, err := store.GetResponse("response-1")
	if err != nil || stored.Status != domain.ResponsePending {
		t.Fatalf("stored: %+v, %v", stored, err)
	}
}
