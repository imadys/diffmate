package suggest

import "testing"

func TestCleanCommitMessage(t *testing.T) {
	message := cleanCommitMessage("\n```txt\nfeat(tui): add codex commit suggestions\n```\n")

	if message != "feat(tui): add codex commit suggestions" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestFirstLineSkipsWarnings(t *testing.T) {
	message := firstLine("WARNING: ignored\nError: codex failed\nextra")

	if message != "Error: codex failed" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestAgentErrorLineSkipsCodexBanner(t *testing.T) {
	message := agentErrorLine("WARNING: ignored\nOpenAI Codex v0.137.0\nerror: model not found\nextra")

	if message != "error: model not found" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestAgentErrorLineSkipsSeparators(t *testing.T) {
	message := agentErrorLine("OpenAI Codex v0.137.0\n--------\nError: failed to initialize")

	if message != "Error: failed to initialize" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestAgentErrorLineIgnoresSourceCode(t *testing.T) {
	message := agentErrorLine("OpenAI Codex v0.137.0\n--------\nmessage = err.Error()\nreturn message")

	if message != "agent command failed; check CLI auth, model, or sandbox settings" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestAgentErrorLineExtractsJSONError(t *testing.T) {
	message := agentErrorLine(`ERROR: {"type":"error","status":400,"error":{"message":"model is not supported"}}`)

	if message != "model is not supported" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestCodexRunnerUsesAccountDefaultModel(t *testing.T) {
	runner, err := runnerForAgent("codex", ".")
	if err != nil {
		t.Fatal(err)
	}

	for i, arg := range runner.args {
		if arg == "--model" || arg == "-m" {
			t.Fatalf("expected codex runner to avoid explicit model, found %q at %d", arg, i)
		}
	}
}

func TestGeminiResponseExtractsJSONResponse(t *testing.T) {
	message := geminiResponse(`{"response":"feat(tui): improve commit modal"}`)

	if message != "feat(tui): improve commit modal" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestRunnerForUnsupportedAgent(t *testing.T) {
	_, err := runnerForAgent("antigravity", ".")
	if err == nil {
		t.Fatal("expected unsupported agent error")
	}
}
