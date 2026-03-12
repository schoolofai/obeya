package tui

import (
	"strings"
	"testing"
	"unicode/utf8"
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
