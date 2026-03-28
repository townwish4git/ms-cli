// Package tools provides executable tools for the agent.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vigo999/ms-cli/integrations/llm"
)

// Tool is the interface for executable tools.
type Tool interface {
	// Name returns the tool name (English, no spaces).
	Name() string

	// Description returns the tool description for LLM understanding.
	Description() string

	// Schema returns the JSON schema for tool parameters.
	Schema() llm.ToolSchema

	// Execute executes the tool with the given parameters.
	Execute(ctx context.Context, params json.RawMessage) (*Result, error)
}

// Result is the result of a tool execution.
type Result struct {
	Content string // Main output content
	Summary string // Summary for UI display (e.g., "42 lines", "5 matches")
	Meta    map[string]any
	Error   error // Execution error
}

// StreamEventType identifies progressive tool output events.
type StreamEventType string

const (
	StreamCmdOutput   StreamEventType = "CmdOutput"
	StreamCmdFinished StreamEventType = "CmdFinished"
)

// StreamEvent is emitted by tools that support progressive output.
type StreamEvent struct {
	Type    StreamEventType
	Message string
}

// StreamSink receives progressive tool output events.
type StreamSink func(StreamEvent)

type streamSinkKey struct{}

// WithStreamSink attaches a progressive output sink to context.
func WithStreamSink(ctx context.Context, sink StreamSink) context.Context {
	if sink == nil {
		return ctx
	}
	return context.WithValue(ctx, streamSinkKey{}, sink)
}

// EmitStreamEvent sends a progressive output event through ctx if configured.
func EmitStreamEvent(ctx context.Context, event StreamEvent) {
	sink, ok := ctx.Value(streamSinkKey{}).(StreamSink)
	if !ok || sink == nil {
		return
	}
	sink(event)
}

// StringResult creates a result with just content.
func StringResult(content string) *Result {
	return &Result{Content: content}
}

// StringResultWithSummary creates a result with content and summary.
func StringResultWithSummary(content, summary string) *Result {
	return &Result{Content: content, Summary: summary}
}

// ErrorResult creates an error result.
func ErrorResult(err error) *Result {
	return &Result{Error: err}
}

// ErrorResultf creates an error result with formatted message.
func ErrorResultf(format string, args ...any) *Result {
	return &Result{Error: fmt.Errorf(format, args...)}
}

// ParseParams parses the raw JSON parameters into a struct.
func ParseParams(data json.RawMessage, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse params: %w", err)
	}
	return nil
}
