package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

func TestFindMarkdownFilesSkipsGeneratedDirs(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "README.md", "# Root\n")
	writeTestFile(t, root, "docs/guide.md", "# Guide\n")
	writeTestFile(t, root, "docs/adr/001.md", "# ADR\n")
	writeTestFile(t, root, "node_modules/pkg/README.md", "# Dependency\n")
	writeTestFile(t, root, "dist/README.md", "# Build output\n")

	files, err := findMarkdownFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(files))
	for _, file := range files {
		got = append(got, file.Path)
	}
	want := []string{"README.md", "docs/adr/001.md", "docs/guide.md"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestDocTreeLinesIncludesParentDirectories(t *testing.T) {
	model := docsModel{
		files: []docFile{
			{Path: "docs/adr/001.md"},
			{Path: "docs/guide.md"},
			{Path: "README.md"},
		},
	}

	lines := model.docTreeLines(80)
	got := make([]string, 0, len(lines))
	for _, line := range lines {
		got = append(got, line.Label)
	}
	want := []string{"▾ docs/", "  ▾ adr/", "    001.md", "  guide.md", "README.md"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestDocTreeLinesHidesCollapsedDirectories(t *testing.T) {
	model := docsModel{
		files: []docFile{
			{Path: "docs/adr/001.md"},
			{Path: "docs/guide.md"},
			{Path: "README.md"},
		},
		collapsed: map[string]bool{"docs": true},
	}

	lines := model.docTreeLines(80)
	got := make([]string, 0, len(lines))
	for _, line := range lines {
		got = append(got, line.Label)
	}
	want := []string{"▸ docs/", "README.md"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestDocsDirtyOnlyWhenEditBufferChanges(t *testing.T) {
	model := docsModel{
		mode:      docsEditMode,
		content:   "hello\nworld",
		editLines: []string{"hello", "world"},
	}
	if model.docsDirty() {
		t.Fatal("expected clean edit buffer")
	}

	model.editLines[1] = "diffmate"
	if !model.docsDirty() {
		t.Fatal("expected dirty edit buffer")
	}
}

func TestNormalizeNewDocPath(t *testing.T) {
	cases := map[string]string{
		"notes":             "notes.md",
		"docs/new file":     "docs/new file.md",
		"/docs/guide.md":    "docs/guide.md",
		"docs/../README.md": "README.md",
	}
	for input, want := range cases {
		got, err := normalizeNewDocPath(input)
		if err != nil {
			t.Fatalf("expected %q to normalize: %v", input, err)
		}
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}

	for _, input := range []string{"", "../secret.md", "notes.txt"} {
		if _, err := normalizeNewDocPath(input); err == nil {
			t.Fatalf("expected %q to fail", input)
		}
	}
}

func TestCreateNewDocFileSelectsAndStartsEdit(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "README.md", "# Root\n")
	model := NewDocs(root).(docsModel)
	model.newFileInput = "docs/new-note"

	if err := model.createNewDocFile(); err != nil {
		t.Fatal(err)
	}

	if model.mode != docsEditMode {
		t.Fatalf("expected edit mode, got %v", model.mode)
	}
	if model.files[model.selected].Path != "docs/new-note.md" {
		t.Fatalf("expected new file selected, got %s", model.files[model.selected].Path)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "new-note.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# New Note\n" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestDeleteEditSelectionAcrossLines(t *testing.T) {
	model := docsModel{
		editLines:  []string{"hello world", "second line"},
		cursorLine: 1,
		cursorCol:  6,
		selecting:  true,
		selectLine: 0,
		selectCol:  6,
	}

	if !model.deleteEditSelection() {
		t.Fatal("expected selection to be deleted")
	}
	if len(model.editLines) != 1 || model.editLines[0] != "hello  line" {
		t.Fatalf("unexpected edit lines: %#v", model.editLines)
	}
	if model.cursorLine != 0 || model.cursorCol != 6 {
		t.Fatalf("unexpected cursor: %d:%d", model.cursorLine, model.cursorCol)
	}
}

func TestNewFileInputEditsAtCursor(t *testing.T) {
	model := docsModel{
		newFileInput:  "docs/new.md",
		newFileCursor: len([]rune("docs/new")),
	}

	model.insertNewFileInput("-note")
	if model.newFileInput != "docs/new-note.md" {
		t.Fatalf("unexpected input: %q", model.newFileInput)
	}

	model.backspaceNewFileInput()
	if model.newFileInput != "docs/new-not.md" {
		t.Fatalf("unexpected input after backspace: %q", model.newFileInput)
	}
}

func TestFormattedDocContentWrapsLongLines(t *testing.T) {
	model := docsModel{
		files:   []docFile{{Path: "README.md"}},
		content: "A buyer deploys SimpleCreator, uploads courses and PDFs, connects Stripe, and starts selling.",
	}

	lines := model.formattedDocContent(28)
	if len(lines) < 3 {
		t.Fatalf("expected wrapped lines, got %#v", lines)
	}
	for _, line := range lines {
		if lipgloss.Width(line) > 28 {
			t.Fatalf("expected line to fit width, got %d for %q", lipgloss.Width(line), line)
		}
	}
	if strings.Contains(strings.Join(lines, "\n"), "…") {
		t.Fatalf("expected wrapped content without truncation, got %#v", lines)
	}
}

func TestWrapLineHardWrapsLongTokens(t *testing.T) {
	lines := wrapLine("supercalifragilistic", 5)
	if len(lines) < 2 {
		t.Fatalf("expected long token to wrap, got %#v", lines)
	}
	for _, line := range lines {
		if lipgloss.Width(line) > 5 {
			t.Fatalf("expected line to fit width, got %d for %q", lipgloss.Width(line), line)
		}
	}
}

func TestDocsEditUsesFullBufferAfterScrolledPreview(t *testing.T) {
	model := docsModel{
		files:   []docFile{{Path: "README.md"}},
		content: "line 0\nline 1\nline 2\nline 3\nline 4\nline 5\nline 6",
	}
	model.viewport = viewport.New(80, 3)
	model.viewport.SetContent(model.content)
	model.viewport.SetYOffset(4)

	model.startDocsEdit()

	if model.cursorLine != 4 {
		t.Fatalf("expected edit cursor to start on scrolled line, got %d", model.cursorLine)
	}
	if model.cursorCol != 0 {
		t.Fatalf("expected edit cursor to start at preview column, got %d", model.cursorCol)
	}
	lines := model.formattedEditContent(80)
	if len(lines) != 7 {
		t.Fatalf("expected full edit buffer for viewport scrolling, got %d lines", len(lines))
	}
	if model.viewport.YOffset != 4 {
		t.Fatalf("expected viewport offset to stay on cursor line, got %d", model.viewport.YOffset)
	}
}

func TestKeepEditCursorVisibleUpdatesViewportOffset(t *testing.T) {
	model := docsModel{
		mode:       docsEditMode,
		editLines:  []string{"0", "1", "2", "3", "4", "5"},
		cursorLine: 5,
	}
	model.viewport = viewport.New(80, 3)

	model.keepEditCursorVisible()

	if model.viewport.YOffset != 3 {
		t.Fatalf("expected viewport offset 3, got %d", model.viewport.YOffset)
	}
}

func writeTestFile(t *testing.T, root, path, content string) {
	t.Helper()

	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
