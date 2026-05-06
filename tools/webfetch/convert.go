package webfetch

import (
	"html"
	"regexp"
	"strings"
)

const maxContentLen = 15000

var (
	reScript     = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle      = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reComment    = regexp.MustCompile(`(?s)<!--.*?-->`)
	reTitle      = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	reMetaDesc   = regexp.MustCompile(`(?is)<meta[^>]*name=["']description["'][^>]*content=["'](.*?)["']`)
	reMetaDescR  = regexp.MustCompile(`(?is)<meta[^>]*content=["'](.*?)["'][^>]*name=["']description["']`)
	reArticle    = regexp.MustCompile(`(?is)<article[^>]*>(.*?)</article>`)
	reMain       = regexp.MustCompile(`(?is)<main[^>]*>(.*?)</main>`)
	reParagraphs = regexp.MustCompile(`(?is)<p[^>]*>(.*?)</p>`)
	reBody       = regexp.MustCompile(`(?is)<body[^>]*>(.*?)</body>`)
	reTags       = regexp.MustCompile(`<[^>]+>`)
	reSpaces     = regexp.MustCompile(`\s+`)

	reHeading = regexp.MustCompile(`(?is)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
	reAnchor  = regexp.MustCompile(`(?is)<a[^>]*href=["']([^"']+)["'][^>]*>(.*?)</a>`)
	reLi      = regexp.MustCompile(`(?is)<li[^>]*>(.*?)</li>`)
	rePre     = regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	reCode    = regexp.MustCompile(`(?is)<code[^>]*>(.*?)</code>`)
	reBr      = regexp.MustCompile(`(?is)<br\s*/?>`)
	reP       = regexp.MustCompile(`(?is)</?p[^>]*>`)
	reBlockQ  = regexp.MustCompile(`(?is)<blockquote[^>]*>(.*?)</blockquote>`)
	reStrong  = regexp.MustCompile(`(?is)<(?:strong|b)[^>]*>(.*?)</(?:strong|b)>`)
	reEm      = regexp.MustCompile(`(?is)<(?:em|i)[^>]*>(.*?)</(?:em|i)>`)
)

// ExtractTitle extracts the content of the <title> tag.
func ExtractTitle(htmlStr string) string {
	m := reTitle.FindStringSubmatch(htmlStr)
	if m == nil {
		return ""
	}
	return cleanText(m[1])
}

// ExtractMetaDescription extracts the meta description content.
func ExtractMetaDescription(htmlStr string) string {
	m := reMetaDesc.FindStringSubmatch(htmlStr)
	if m == nil {
		m = reMetaDescR.FindStringSubmatch(htmlStr)
	}
	if m == nil {
		return ""
	}
	return html.UnescapeString(strings.TrimSpace(m[1]))
}

// ExtractContent extracts the main text content from HTML.
func ExtractContent(htmlStr string) string {
	text := reScript.ReplaceAllString(htmlStr, "")
	text = reStyle.ReplaceAllString(text, "")
	text = reComment.ReplaceAllString(text, "")

	for _, re := range []*regexp.Regexp{reArticle, reMain} {
		matches := re.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			cleaned := cleanText(m[1])
			if len(cleaned) > 200 {
				return capStr(cleaned, maxContentLen)
			}
		}
	}

	pMatches := reParagraphs.FindAllStringSubmatch(text, -1)
	if len(pMatches) > 0 {
		var parts []string
		for _, m := range pMatches {
			p := cleanText(m[1])
			if len(p) > 30 {
				parts = append(parts, p)
			}
		}
		joined := strings.Join(parts, " ")
		if len(joined) > 100 {
			return capStr(joined, maxContentLen)
		}
	}

	if m := reBody.FindStringSubmatch(text); m != nil {
		cleaned := cleanText(m[1])
		return capStr(cleaned, maxContentLen)
	}

	return ""
}

// HtmlToMarkdown converts HTML to a simplified Markdown representation.
func HtmlToMarkdown(htmlStr string) string {
	text := reScript.ReplaceAllString(htmlStr, "")
	text = reStyle.ReplaceAllString(text, "")
	text = reComment.ReplaceAllString(text, "")

	content := ""
	for _, re := range []*regexp.Regexp{reArticle, reMain} {
		if m := re.FindStringSubmatch(text); m != nil {
			if len(stripTags(m[1])) > 200 {
				content = m[1]
				break
			}
		}
	}
	if content == "" {
		if m := reBody.FindStringSubmatch(text); m != nil {
			content = m[1]
		} else {
			content = text
		}
	}

	content = rePre.ReplaceAllStringFunc(content, func(s string) string {
		m := rePre.FindStringSubmatch(s)
		if m == nil {
			return s
		}
		inner := reTags.ReplaceAllString(m[1], "")
		return "\n```\n" + html.UnescapeString(strings.TrimSpace(inner)) + "\n```\n"
	})

	content = reHeading.ReplaceAllStringFunc(content, func(s string) string {
		m := reHeading.FindStringSubmatch(s)
		if m == nil {
			return s
		}
		level := m[1][0] - '0'
		prefix := strings.Repeat("#", int(level))
		return "\n" + prefix + " " + stripTags(m[2]) + "\n"
	})

	content = reAnchor.ReplaceAllStringFunc(content, func(s string) string {
		m := reAnchor.FindStringSubmatch(s)
		if m == nil {
			return s
		}
		linkText := stripTags(m[2])
		if linkText == "" {
			linkText = m[1]
		}
		return "[" + strings.TrimSpace(linkText) + "](" + m[1] + ")"
	})

	content = reStrong.ReplaceAllString(content, "**$1**")
	content = reEm.ReplaceAllString(content, "*$1*")
	content = reCode.ReplaceAllString(content, "`$1`")

	content = reLi.ReplaceAllStringFunc(content, func(s string) string {
		m := reLi.FindStringSubmatch(s)
		if m == nil {
			return s
		}
		return "\n- " + strings.TrimSpace(stripTags(m[1]))
	})

	content = reBlockQ.ReplaceAllStringFunc(content, func(s string) string {
		m := reBlockQ.FindStringSubmatch(s)
		if m == nil {
			return s
		}
		inner := strings.TrimSpace(stripTags(m[1]))
		lines := strings.Split(inner, "\n")
		for i, l := range lines {
			lines[i] = "> " + l
		}
		return "\n" + strings.Join(lines, "\n") + "\n"
	})

	content = reBr.ReplaceAllString(content, "\n")
	content = reP.ReplaceAllString(content, "\n\n")
	content = reTags.ReplaceAllString(content, "")
	content = html.UnescapeString(content)

	lines := strings.Split(content, "\n")
	var result []string
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, "")
			}
		} else {
			blankCount = 0
			result = append(result, trimmed)
		}
	}

	final := strings.TrimSpace(strings.Join(result, "\n"))
	return capStr(final, maxContentLen)
}

func cleanText(s string) string {
	s = reTags.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func stripTags(s string) string {
	return strings.TrimSpace(reTags.ReplaceAllString(s, ""))
}

func capStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
