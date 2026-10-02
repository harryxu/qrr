package runner

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("QRR_TEST_PROCESS") != "1" {
		return
	}
	if os.Args[len(os.Args)-1] == "wait" {
		time.Sleep(30 * time.Second)
		os.Exit(0)
	}
	os.Exit(23)
}
func TestExitCode(t *testing.T) {
	t.Setenv("QRR_TEST_PROCESS", "1")
	err := Run(context.Background(), []string{os.Args[0], "-test.run=TestHelperProcess", "--", "exit"})
	if ExitCode(err) != 23 {
		t.Fatalf("got %v / %d", err, ExitCode(err))
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatal("expected process exit error")
	}
}
func TestCancellation(t *testing.T) {
	t.Setenv("QRR_TEST_PROCESS", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := Run(ctx, []string{os.Args[0], "-test.run=TestHelperProcess", "--", "wait"})
	if err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("cancellation failed: %v", err)
	}
}
