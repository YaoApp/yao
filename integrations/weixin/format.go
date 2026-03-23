package weixin

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// FormatWeixinText converts standard Markdown to plain text suitable for
// WeChat iLink Bot's text_item. WeChat renders only plain text with clickable
// URLs and [text](url) style links. All other Markdown/HTML is stripped and
// gracefully degraded to readable plain text.
func FormatWeixinText(md string) string {
	md = strings.ReplaceAll(md, "\r\n", "\n")

	var out strings.Builder
	lines := strings.Split(md, "\n")

	inCodeBlock := false
	var codeLines []string

	inTable := false
	var tableRows [][]string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				codeLines = nil
			} else {
				inCodeBlock = false
				for _, cl := range codeLines {
					out.WriteString("  " + cl + "\n")
				}
			}
			continue
		}
		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		if wxIsTableRow(line) {
			if !inTable {
				inTable = true
				tableRows = nil
			}
			if wxIsTableSep(line) {
				continue
			}
			tableRows = append(tableRows, wxParseTableRow(line))
			continue
		}
		if inTable {
			wxFlushTable(&out, tableRows)
			inTable = false
			tableRows = nil
		}

		if line == "---" || line == "***" || line == "___" {
			out.WriteString("——————\n")
			continue
		}

		if m := wxReHeading.FindStringSubmatch(line); m != nil {
			out.WriteString("【" + wxFormatInline(m[2]) + "】\n")
			continue
		}

		if m := wxReBlockquote.FindStringSubmatch(line); m != nil {
			out.WriteString("│ " + wxFormatInline(m[1]) + "\n")
			continue
		}

		if m := wxReUnorderedList.FindStringSubmatch(line); m != nil {
			out.WriteString("• " + wxFormatInline(m[1]) + "\n")
			continue
		}

		if m := wxReOrderedList.FindStringSubmatch(line); m != nil {
			out.WriteString(m[1] + ". " + wxFormatInline(m[2]) + "\n")
			continue
		}

		if m := wxReImage.FindStringSubmatch(line); m != nil {
			alt := m[1]
			url := m[2]
			if alt != "" {
				out.WriteString("[" + alt + "](" + url + ")\n")
			} else {
				out.WriteString(url + "\n")
			}
			continue
		}

		out.WriteString(wxFormatInline(line) + "\n")
	}

	if inCodeBlock && len(codeLines) > 0 {
		for _, cl := range codeLines {
			out.WriteString("  " + cl + "\n")
		}
	}
	if inTable {
		wxFlushTable(&out, tableRows)
	}

	return strings.TrimRight(out.String(), "\n")
}

var (
	wxReHeading       = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	wxReBlockquote    = regexp.MustCompile(`^>\s*(.*)$`)
	wxReUnorderedList = regexp.MustCompile(`^[\s]*[-*+]\s+(.+)$`)
	wxReOrderedList   = regexp.MustCompile(`^[\s]*(\d+)[.)]\s+(.+)$`)
	wxReImage         = regexp.MustCompile(`^!\[([^\]]*)\]\(([^)]+)\)$`)
	wxReTableRow      = regexp.MustCompile(`^\|.*\|$`)
	wxReTableSep      = regexp.MustCompile(`^\|[\s\-:|]+\|$`)

	wxReBoldItalic    = regexp.MustCompile(`\*\*\*(.+?)\*\*\*`)
	wxReBold          = regexp.MustCompile(`\*\*(.+?)\*\*`)
	wxReBoldAlt       = regexp.MustCompile(`__(.+?)__`)
	wxReItalic        = regexp.MustCompile(`(?:^|[^*])\*([^*]+?)\*(?:[^*]|$)`)
	wxReStrikethrough = regexp.MustCompile(`~~(.+?)~~`)
	wxReCode          = regexp.MustCompile("`([^`]+)`")
	wxReLink          = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)

	wxReHTMLTag = regexp.MustCompile(`<[^>]+>`)
)

// wxFormatInline strips inline Markdown/HTML formatting, keeping links in
// [text](url) form which WeChat renders as clickable.
func wxFormatInline(s string) string {
	s = wxReLink.ReplaceAllString(s, "[$1]($2)")
	s = wxReBoldItalic.ReplaceAllString(s, "$1")
	s = wxReBold.ReplaceAllString(s, "$1")
	s = wxReBoldAlt.ReplaceAllString(s, "$1")
	s = wxReStrikethrough.ReplaceAllString(s, "$1")
	s = wxReCode.ReplaceAllString(s, "$1")
	s = wxReItalic.ReplaceAllStringFunc(s, func(match string) string {
		m := wxReItalic.FindStringSubmatch(match)
		if len(m) < 2 {
			return match
		}
		prefix := ""
		suffix := ""
		if len(match) > 0 && match[0] != '*' {
			prefix = string(match[0])
		}
		if len(match) > 0 && match[len(match)-1] != '*' {
			suffix = string(match[len(match)-1])
		}
		return prefix + m[1] + suffix
	})
	s = wxReHTMLTag.ReplaceAllString(s, "")
	return s
}

func wxIsTableRow(line string) bool {
	return wxReTableRow.MatchString(strings.TrimSpace(line))
}

func wxIsTableSep(line string) bool {
	return wxReTableSep.MatchString(strings.TrimSpace(line))
}

func wxParseTableRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	cells := strings.Split(line, "|")
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

func wxFlushTable(out *strings.Builder, rows [][]string) {
	if len(rows) == 0 {
		return
	}
	colWidths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && utf8.RuneCountInString(cell) > colWidths[i] {
				colWidths[i] = utf8.RuneCountInString(cell)
			}
		}
	}
	for ri, row := range rows {
		for ci, cell := range row {
			if ci > 0 {
				out.WriteString(" | ")
			}
			w := 0
			if ci < len(colWidths) {
				w = colWidths[ci]
			}
			out.WriteString(wxPadRight(cell, w))
		}
		out.WriteString("\n")
		if ri == 0 && len(rows) > 1 {
			for ci := range row {
				if ci > 0 {
					out.WriteString("-+-")
				}
				w := 0
				if ci < len(colWidths) {
					w = colWidths[ci]
				}
				out.WriteString(strings.Repeat("-", w))
			}
			out.WriteString("\n")
		}
	}
}

func wxPadRight(s string, width int) string {
	runes := utf8.RuneCountInString(s)
	if runes >= width {
		return s
	}
	return s + strings.Repeat(" ", width-runes)
}
