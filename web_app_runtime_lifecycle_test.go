package main

import (
	"errors"
	"testing"
	"time"
)

func TestBrowserLauncherDetachedOnlyForQuickSuccessfulExit(t *testing.T) {
	tests := []struct {
		name     string
		waitErr  error
		lifetime time.Duration
		want     bool
	}{
		{name: "quick handoff", lifetime: time.Second, want: true},
		{name: "launch failure", waitErr: errors.New("failed"), lifetime: time.Second},
		{name: "normal browser close", lifetime: browserLauncherDetachWindow},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := browserLauncherDetached(test.waitErr, test.lifetime); got != test.want {
				t.Fatalf("browserLauncherDetached() = %v, want %v", got, test.want)
			}
		})
	}
}
