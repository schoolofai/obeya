package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/niladribose/obeya/internal/domain"
)

func TestWrapText_ShortString(t *testing.T) {
	lines := wrapText("hello", 20)
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [\"hello\"], got %v", lines)
	}
}

func TestWrapText_ExactWidth(t *testing.T) {
	lines := wrapText("hello", 5)
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("expected [\"hello\"], got %v", lines)
	}
}

func TestWrapText_WordBoundary(t *testing.T) {
	lines := wrapText("hello world foo", 11)
	if len(lines) != 2 || lines[0] != "hello world" || lines[1] != "foo" {
		t.Errorf("expected [\"hello world\", \"foo\"], got %v", lines)
	}
}

func TestWrapText_LongWord(t *testing.T) {
	lines := wrapText("abcdefghij", 5)
	if len(lines) != 2 || lines[0] != "abcde" || lines[1] != "fghij" {
		t.Errorf("expected [\"abcde\", \"fghij\"], got %v", lines)
	}
}

func TestWrapText_MultipleWraps(t *testing.T) {
	lines := wrapText("one two three four five", 9)
	// "one two" (7), "three" (5), "four five" (9)
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %v", len(lines), lines)
	}
}

func TestWrapText_Empty(t *testing.T) {
	lines := wrapText("", 20)
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("expected [\"\"], got %v", lines)
	}
}

func TestWrapText_SingleSpaces(t *testing.T) {
	// Multiple spaces between words should not produce empty tokens
	lines := wrapText("a  b  c", 10)
	joined := strings.Join(lines, " ")
	if !strings.Contains(joined, "a") || !strings.Contains(joined, "b") || !strings.Contains(joined, "c") {
		t.Errorf("expected all words present, got %v", lines)
	}
}

func TestWrapText_Unicode(t *testing.T) {
	// Multi-byte characters should wrap at rune boundaries, not byte boundaries
	lines := wrapText("héllo wörld", 7)
	for i, line := range lines {
		if utf8.RuneCountInString(line) > 7 {
			t.Errorf("line %d exceeds maxWidth in runes: %q (%d runes)", i, line, utf8.RuneCountInString(line))
		}
	}
	joined := strings.Join(lines, " ")
	if joined != "héllo wörld" {
		t.Errorf("expected original text reconstructed, got %q", joined)
	}
}

func TestWrapText_TitleLikeString(t *testing.T) {
	title := "Implement automatic token refresh handling"
	lines := wrapText(title, 16)
	joined := strings.Join(lines, " ")
	if joined != title {
		t.Errorf("wrapped lines should reconstruct title\nexpected: %q\ngot:      %q", title, joined)
	}
	for i, line := range lines {
		if utf8.RuneCountInString(line) > 16 {
			t.Errorf("line %d exceeds maxWidth: %q (%d runes)", i, line, utf8.RuneCountInString(line))
		}
	}
}

// --- renderDescription tests ---

func TestRenderDescription_Short(t *testing.T) {
	a := App{}
	lines := a.renderDescription("Short desc", 20, 0, 5)
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d: %v", len(lines), lines)
	}
}

func TestRenderDescription_ExactlyMaxLines(t *testing.T) {
	a := App{}
	desc := "line one\nline two\nline three\nline four\nline five"
	lines := a.renderDescription(desc, 40, 0, 5)
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d: %v", len(lines), lines)
	}
}

func TestRenderDescription_OverMaxLines_ShowsScrollDown(t *testing.T) {
	a := App{}
	desc := "one\ntwo\nthree\nfour\nfive\nsix\nseven"
	lines := a.renderDescription(desc, 40, 0, 5)
	// 5 content lines + 1 scroll indicator
	if len(lines) != 6 {
		t.Errorf("expected 6 lines (5 content + 1 indicator), got %d: %v", len(lines), lines)
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "\u25be") { // ▾
		t.Errorf("expected down scroll indicator, got: %q", last)
	}
}

func TestRenderDescription_ScrolledMiddle_ShowsBothIndicators(t *testing.T) {
	a := App{}
	desc := "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight"
	lines := a.renderDescription(desc, 40, 2, 5)
	// 1 up indicator + 5 content + 1 down indicator
	if len(lines) != 7 {
		t.Errorf("expected 7 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "\u25b4") { // ▴
		t.Errorf("expected up scroll indicator on first line, got: %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "\u25be") { // ▾
		t.Errorf("expected down scroll indicator on last line, got: %q", lines[len(lines)-1])
	}
}

func TestRenderDescription_ScrolledToEnd_ShowsUpOnly(t *testing.T) {
	a := App{}
	desc := "one\ntwo\nthree\nfour\nfive\nsix\nseven"
	// 7 lines total, scrollY=2 means lines 2-6 visible (indices 2,3,4,5,6)
	lines := a.renderDescription(desc, 40, 2, 5)
	if !strings.Contains(lines[0], "\u25b4") { // ▴
		t.Errorf("expected up indicator, got: %q", lines[0])
	}
	last := lines[len(lines)-1]
	if strings.Contains(last, "\u25be") { // ▾
		t.Errorf("should NOT have down indicator when scrolled to end, got: %q", last)
	}
}

func TestRenderDescription_Empty(t *testing.T) {
	a := App{}
	lines := a.renderDescription("", 20, 0, 5)
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for empty desc, got %d", len(lines))
	}
}

func TestRenderDescription_WrapsLongLines(t *testing.T) {
	a := App{}
	desc := "This is a very long description line that should be wrapped to fit"
	lines := a.renderDescription(desc, 15, 0, 5)
	for i, line := range lines {
		plain := stripAnsi(line)
		if utf8.RuneCountInString(plain) > 15 {
			t.Errorf("line %d exceeds maxWidth: %q (%d runes)", i, plain, utf8.RuneCountInString(plain))
		}
	}
}

// stripAnsi removes ANSI escape sequences for length testing.
func stripAnsi(s string) string {
	result := ""
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		result += string(r)
	}
	return result
}

// --- renderCard assignee badge tests ---

func newTestApp(items map[string]*domain.Item, users map[string]*domain.Identity) App {
	board := &domain.Board{
		Items: items,
		Users: users,
	}
	return App{board: board}
}

func TestRenderCard_UnassignedBadge(t *testing.T) {
	item := &domain.Item{
		ID:         "t1",
		DisplayNum: 1,
		Title:      "Fix login bug",
		Type:       domain.ItemTypeTask,
		Priority:   domain.PriorityMedium,
	}
	a := newTestApp(
		map[string]*domain.Item{"t1": item},
		map[string]*domain.Identity{},
	)

	card := a.renderCard(item, false)
	plain := stripAnsi(card)
	if !strings.Contains(plain, "@unassigned") {
		t.Errorf("expected @unassigned badge, got:\n%s", plain)
	}
}

func TestRenderCard_AssignedBadge(t *testing.T) {
	item := &domain.Item{
		ID:         "t2",
		DisplayNum: 2,
		Title:      "Add tests",
		Type:       domain.ItemTypeTask,
		Priority:   domain.PriorityHigh,
		Assignee:   "user-1",
	}
	a := newTestApp(
		map[string]*domain.Item{"t2": item},
		map[string]*domain.Identity{
			"user-1": {ID: "user-1", Name: "alice"},
		},
	)

	card := a.renderCard(item, false)
	plain := stripAnsi(card)
	if !strings.Contains(plain, "@alice") {
		t.Errorf("expected @alice badge, got:\n%s", plain)
	}
	if strings.Contains(plain, "@unassigned") {
		t.Errorf("should not show @unassigned when assigned, got:\n%s", plain)
	}
}

func TestRenderCard_UnassignedNotShownWhenAssigned(t *testing.T) {
	item := &domain.Item{
		ID:         "t3",
		DisplayNum: 3,
		Title:      "Deploy service",
		Type:       domain.ItemTypeStory,
		Priority:   domain.PriorityLow,
		Assignee:   "user-2",
	}
	a := newTestApp(
		map[string]*domain.Item{"t3": item},
		map[string]*domain.Identity{
			"user-2": {ID: "user-2", Name: "bob"},
		},
	)

	card := a.renderCard(item, false)
	plain := stripAnsi(card)
	if strings.Contains(plain, "@unassigned") {
		t.Errorf("@unassigned should not appear on assigned card, got:\n%s", plain)
	}
	if !strings.Contains(plain, "@bob") {
		t.Errorf("expected @bob, got:\n%s", plain)
	}
}

// --- firstRegisteredUser tests ---

func TestFirstRegisteredUser_WithUsers(t *testing.T) {
	a := newTestApp(
		map[string]*domain.Item{},
		map[string]*domain.Identity{
			"user-1": {ID: "user-1", Name: "Alice"},
		},
	)
	got := a.firstRegisteredUser()
	if got == "" {
		t.Fatal("expected a user ID, got empty string")
	}
	if got != "user-1" {
		t.Errorf("expected 'user-1', got %q", got)
	}
}

func TestFirstRegisteredUser_NoUsers(t *testing.T) {
	a := newTestApp(
		map[string]*domain.Item{},
		map[string]*domain.Identity{},
	)
	got := a.firstRegisteredUser()
	if got != "" {
		t.Errorf("expected empty string when no users, got %q", got)
	}
}

func TestFirstRegisteredUser_NilBoard(t *testing.T) {
	a := App{board: nil}
	got := a.firstRegisteredUser()
	if got != "" {
		t.Errorf("expected empty string with nil board, got %q", got)
	}
}
