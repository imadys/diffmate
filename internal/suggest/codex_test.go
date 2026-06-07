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
