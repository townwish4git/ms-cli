package loop

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vigo999/ms-cli/integrations/llm"
	"github.com/vigo999/ms-cli/tools"
)

type streamingStubTool struct{}

func (streamingStubTool) Name() string { return "shell" }

func (streamingStubTool) Description() string { return "streaming shell stub" }

func (streamingStubTool) Schema() llm.ToolSchema { return llm.ToolSchema{Type: "object"} }

func (streamingStubTool) Execute(ctx context.Context, _ json.RawMessage) (*tools.Result, error) {
	tools.EmitStreamEvent(ctx, tools.StreamEvent{
		Type:    tools.StreamCmdOutput,
		Message: "loss=0.42",
	})
	tools.EmitStreamEvent(ctx, tools.StreamEvent{
		Type:    tools.StreamCmdFinished,
		Message: "exit 0",
	})
	return tools.StringResultWithSummary("loss=0.42", "completed"), nil
}

func TestRunWithContextStream_EmitsCmdOutputAndCmdFinishedForStreamingShell(t *testing.T) {
	args, err := json.Marshal(map[string]any{
		"command":       "python train.py",
		"stream_output": true,
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	provider := &scriptedStreamProvider{
		responses: []*llm.CompletionResponse{
			{
				ToolCalls: []llm.ToolCall{{
					ID:   "call-shell-1",
					Type: "function",
					Function: llm.ToolCallFunc{
						Name:      "shell",
						Arguments: args,
					},
				}},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Content:      "done",
				FinishReason: llm.FinishStop,
			},
		},
	}

	registry := tools.NewRegistry()
	registry.MustRegister(streamingStubTool{})

	engine := NewEngine(EngineConfig{
		MaxIterations: 2,
		ContextWindow: 4096,
	}, provider, registry)

	var gotOutput, gotFinished bool
	err = engine.RunWithContextStream(context.Background(), Task{
		ID:          "stream-shell",
		Description: "run shell",
	}, func(ev Event) {
		if ev.Type == EventCmdOutput && ev.Message == "loss=0.42" {
			gotOutput = true
		}
		if ev.Type == EventCmdFinished && ev.Message == "exit 0" {
			gotFinished = true
		}
	})
	if err != nil {
		t.Fatalf("run stream: %v", err)
	}
	if !gotOutput {
		t.Fatalf("expected CmdOutput event")
	}
	if !gotFinished {
		t.Fatalf("expected CmdFinished event")
	}
}
