package tui

import "strings"

// PickerModel presents a list of choices for the user to select from.
type PickerModel struct {
	title  string
	kind   pickerKind
	items  []string
	cursor int
}

func newPickerModel(title string, kind pickerKind, items []string) PickerModel {
	return PickerModel{title: title, kind: kind, items: items}
}

func (p PickerModel) View() string {
	var b strings.Builder
	b.WriteString(p.title)
	b.WriteString("\n\n")
	for i, item := range p.items {
		if i == p.cursor {
			b.WriteString("  ▸ " + item + "\n")
		} else {
			b.WriteString("    " + item + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Enter:select  Esc:cancel"))
	return overlayStyle.Render(b.String())
}

func (p *PickerModel) MoveUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *PickerModel) MoveDown() {
	if p.cursor < len(p.items)-1 {
		p.cursor++
	}
}

func (p PickerModel) Selected() string {
	if len(p.items) == 0 {
		return ""
	}
	return p.items[p.cursor]
}
