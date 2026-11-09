package remotessh

import (
	"context"
	"errors"
	"testing"
	"time"
)

type keepAliveSender struct {
	calls   chan struct{}
	results chan error
}

func (sender *keepAliveSender) SendRequest(string, bool, []byte) (bool, []byte, error) {
	sender.calls <- struct{}{}
	err := <-sender.results
	return err == nil, nil, err
}

func TestMonitorKeepAliveReportsConsecutiveFailureThreshold(t *testing.T) {
	want := errors.New("connection lost")
	sender := &keepAliveSender{
		calls:   make(chan struct{}),
		results: make(chan error),
	}
	failure := make(chan error, 1)

	go MonitorKeepAlive(context.Background(), sender, KeepAliveOptions{
		Interval:         time.Millisecond,
		FailureThreshold: 2,
	}, func(err error) {
		failure <- err
	})

	answerKeepAlive(t, sender, want)
	answerKeepAlive(t, sender, want)
	select {
	case got := <-failure:
		if !errors.Is(got, want) {
			t.Fatalf("failure = %v, want %v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("MonitorKeepAlive did not report the failure")
	}
}

func TestMonitorKeepAliveResetsFailureCountAfterSuccess(t *testing.T) {
	want := errors.New("connection lost")
	sender := &keepAliveSender{
		calls:   make(chan struct{}),
		results: make(chan error),
	}
	failure := make(chan error, 1)

	go MonitorKeepAlive(context.Background(), sender, KeepAliveOptions{
		Interval:         time.Millisecond,
		FailureThreshold: 2,
	}, func(err error) {
		failure <- err
	})

	answerKeepAlive(t, sender, want)
	answerKeepAlive(t, sender, nil)
	answerKeepAlive(t, sender, want)
	select {
	case err := <-failure:
		t.Fatalf("failure reported before two consecutive errors: %v", err)
	default:
	}

	answerKeepAlive(t, sender, want)
	select {
	case got := <-failure:
		if !errors.Is(got, want) {
			t.Fatalf("failure = %v, want %v", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("MonitorKeepAlive did not report consecutive failures")
	}
}

func TestMonitorKeepAliveStopsWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sender := &keepAliveSender{calls: make(chan struct{}, 1), results: make(chan error)}
	failure := make(chan error, 1)
	done := make(chan struct{})

	go func() {
		MonitorKeepAlive(ctx, sender, KeepAliveOptions{Interval: time.Millisecond}, func(err error) {
			failure <- err
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("MonitorKeepAlive did not stop after cancellation")
	}
	select {
	case <-sender.calls:
		t.Fatal("keepalive request sent after cancellation")
	case err := <-failure:
		t.Fatalf("failure reported after cancellation: %v", err)
	default:
	}
}

func answerKeepAlive(t *testing.T, sender *keepAliveSender, err error) {
	t.Helper()
	select {
	case <-sender.calls:
	case <-time.After(time.Second):
		t.Fatal("MonitorKeepAlive did not send the expected request")
	}
	sender.results <- err
}
