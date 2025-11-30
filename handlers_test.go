package main

import (
	"context"
	"testing"
	"time"
)

func TestEnsureConversionWorkersReinitialize(t *testing.T) {
	t.Parallel()

	ctx1, cancel1 := context.WithCancel(context.Background())
	queue1 := ensureConversionWorkers(ctx1)
	if queue1 == nil {
		t.Fatalf("expected non-nil queue")
	}
	cancel1()

	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("queue1 was not closed after context cancellation")
	case _, ok := <-queue1:
		if ok {
			t.Fatalf("queue1 should be closed")
		}
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	queue2 := ensureConversionWorkers(ctx2)
	if queue2 == nil {
		t.Fatalf("expected non-nil queue on second init")
	}
	if queue2 == queue1 {
		t.Fatalf("expected new queue after previous shutdown")
	}

	cancel2()
	select {
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("queue2 was not closed after context cancellation")
	case _, ok := <-queue2:
		if ok {
			t.Fatalf("queue2 should be closed")
		}
	}
}
