package remotessh

import (
	"context"
	"time"
)

const (
	DefaultKeepAliveInterval         = 30 * time.Second
	DefaultKeepAliveFailureThreshold = 2
)

type RequestSender interface {
	SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error)
}

type KeepAliveOptions struct {
	Interval         time.Duration
	FailureThreshold int
}

func MonitorKeepAlive(ctx context.Context, sender RequestSender, options KeepAliveOptions, onFailure func(error)) {
	interval := options.Interval
	if interval <= 0 {
		interval = DefaultKeepAliveInterval
	}
	failureThreshold := options.FailureThreshold
	if failureThreshold <= 0 {
		failureThreshold = DefaultKeepAliveFailureThreshold
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _, err := sender.SendRequest("keepalive@openssh.com", true, nil)
			if err == nil {
				failures = 0
				continue
			}
			failures++
			if failures >= failureThreshold {
				if onFailure != nil {
					onFailure(err)
				}
				return
			}
		}
	}
}
