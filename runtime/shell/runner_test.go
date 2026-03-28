package shell

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunnerRunWithOptions_StreamOutput_EmitsStdoutAndStderrLines(t *testing.T) {
	runner := NewRunner(Config{
		WorkDir: ".",
		Timeout: 3 * time.Second,
	})

	var lines []ShellLine
	result, err := runner.RunWithOptions(context.Background(),
		"printf 'out1\\n'; printf 'err1\\n' 1>&2; printf 'out2\\n'",
		RunOptions{
			StreamOutput: true,
			OnLine: func(line ShellLine) {
				lines = append(lines, line)
			},
		},
	)
	if err != nil {
		t.Fatalf("run with options: %v", err)
	}
	if result.Error != nil {
		t.Fatalf("unexpected result error: %v", result.Error)
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", result.ExitCode)
	}

	var gotStdout, gotStderr bool
	for _, line := range lines {
		if line.Source == "stdout" && strings.Contains(line.Text, "out1") {
			gotStdout = true
		}
		if line.Source == "stderr" && strings.Contains(line.Text, "err1") {
			gotStderr = true
		}
	}
	if !gotStdout {
		t.Fatalf("expected streamed stdout line, got %#v", lines)
	}
	if !gotStderr {
		t.Fatalf("expected streamed stderr line, got %#v", lines)
	}
}

func TestRunnerRunWithOptions_StreamOutput_RespectsCancellationAndKeepsPartial(t *testing.T) {
	runner := NewRunner(Config{
		WorkDir: ".",
		Timeout: 10 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1300*time.Millisecond)
	defer cancel()

	var lines []ShellLine
	result, err := runner.RunWithOptions(ctx,
		"i=1; while [ $i -le 5 ]; do echo tick-$i; i=$((i+1)); sleep 1; done",
		RunOptions{
			StreamOutput: true,
			OnLine: func(line ShellLine) {
				lines = append(lines, line)
			},
		},
	)
	if err != nil {
		t.Fatalf("run with options: %v", err)
	}
	if result == nil {
		t.Fatalf("expected result on cancellation")
	}
	if !errors.Is(result.Error, context.DeadlineExceeded) && !errors.Is(result.Error, context.Canceled) {
		t.Fatalf("expected context cancellation/deadline error, got %v", result.Error)
	}
	if len(lines) == 0 {
		t.Fatalf("expected partial streamed lines before cancellation")
	}
}
