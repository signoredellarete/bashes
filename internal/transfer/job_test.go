package transfer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestManagerAllowsOnlyOneActiveJobPerResource(t *testing.T) {
	manager := NewManager()
	first, _, err := manager.Start(JobOptions{Info: testJobInfo("one", "session-1", "host-1")})
	if err != nil {
		t.Fatalf("Start(first) error = %v", err)
	}
	if _, _, err := manager.Start(JobOptions{Info: testJobInfo("two", "session-2", "host-1")}); err == nil {
		t.Fatal("Start(second) error = nil, want active resource conflict")
	}
	first.Finish(nil)
	if _, _, err := manager.Start(JobOptions{Info: testJobInfo("two", "session-2", "host-1")}); err != nil {
		t.Fatalf("Start(after finish) error = %v", err)
	}
}

func TestJobFinishRecordsTerminalStateAndClosesDone(t *testing.T) {
	manager := NewManager()
	job, ctx, err := manager.Start(JobOptions{Info: testJobInfo("one", "session-1", "host-1")})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	want := errors.New("disk full")
	job.Finish(want)

	if info := job.Snapshot(); info.Status != "failed" || info.Error != want.Error() || info.FinishedAt == "" {
		t.Fatalf("Snapshot() = %+v", info)
	}
	select {
	case <-job.Done():
	default:
		t.Fatal("Done() is not closed")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("context error = %v, want canceled", ctx.Err())
	}
}

func TestManagerCancelAndDeleteSessionJobs(t *testing.T) {
	manager := NewManager()
	job, ctx, err := manager.Start(JobOptions{Info: testJobInfo("one", "session-1", "host-1")})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	jobs := manager.CancelForSession("session-1")
	if len(jobs) != 1 || jobs[0] != job {
		t.Fatalf("CancelForSession() = %v", jobs)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("context error = %v, want canceled", ctx.Err())
	}
	job.Finish(ctx.Err())
	manager.DeleteForSession("session-1")
	if got := manager.InfosForSession("session-1"); len(got) != 0 {
		t.Fatalf("InfosForSession() = %v", got)
	}
}

func TestManagerDismissAndPruneFinishedJobs(t *testing.T) {
	manager := NewManager()
	active, _, err := manager.Start(JobOptions{Info: testJobInfo("active", "session-1", "host-1")})
	if err != nil {
		t.Fatalf("Start(active) error = %v", err)
	}
	if err := manager.Dismiss("active"); err == nil {
		t.Fatal("Dismiss(active) error = nil")
	}
	active.Finish(nil)
	if err := manager.Dismiss("active"); err != nil {
		t.Fatalf("Dismiss(finished) error = %v", err)
	}

	for index := 0; index < 5; index++ {
		id := fmt.Sprintf("job-%d", index)
		job, _, startErr := manager.Start(JobOptions{Info: testJobInfo(id, "session-1", "host-1")})
		if startErr != nil {
			t.Fatalf("Start(%s) error = %v", id, startErr)
		}
		job.Finish(nil)
		job.Update(func(info *JobInfo) {
			info.FinishedAt = time.Unix(int64(index), 0).Format(time.RFC3339)
		})
	}
	manager.PruneFinished("host-1", 2)
	if got := manager.InfosForSession("session-1"); len(got) != 2 {
		t.Fatalf("retained jobs = %d, want 2", len(got))
	}
}

func testJobInfo(id string, sessionID string, resourceID string) JobInfo {
	return JobInfo{JobID: id, SessionID: sessionID, ResourceID: resourceID, Status: "queued"}
}
