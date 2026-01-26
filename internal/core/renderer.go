package core

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"mdbook-gen/internal/config"

	"mdbook-gen/templates"

	"gopkg.in/yaml.v3"
)

// Global configuration
var conf config.Config

// Chapter 表示一个章节
type Chapter struct {
	ID         string
	Number     string // e.g. "2.1."
	Title      string
	InputFile  string
	OutputFile string
	Category   string // e.g. "基础核心"
	Content    template.HTML
	IsContents bool
	IsFront    bool
}

func RenderBook(rootDir string, outputDirOverride string) error {
	// 加载配置
	confPath := filepath.Join(rootDir, "book.yaml")
	confBody, err := os.ReadFile(confPath)
	if err != nil {
		return fmt.Errorf("无法读取 book.yaml (%s): %w", confPath, err)
	}
	if err := yaml.Unmarshal(confBody, &conf); err != nil {
		return fmt.Errorf("解析 book.yaml 失败: %w", err)
	}

	files, _ := filepath.Glob(filepath.Join(rootDir, "book", "*.md"))
	sort.Strings(files)

	var chapters []Chapter
	chapterNum := 0
	subChapter := 0

	for _, file := range files {
		content, _ := os.ReadFile(file)
		filename := filepath.Base(file)
		title := extractTitle(string(content))

		var outFile string
		var number string
		var cat string

		switch filename {
		case "00.00-frontmatter.md":
			outFile = "00.00-front-matter.html"
			title = "前言"
			chapters = append(chapters, Chapter{
				ID:         "00.00-front-matter",
				Title:      title,
				InputFile:  file,
				OutputFile: outFile,
				IsFront:    true,
			})
		case "00.01-contents.md":
			outFile = "00.01-contents.html"
			title = "目录"
			chapters = append(chapters, Chapter{
				ID:         "00.01-contents",
				Title:      title,
				InputFile:  file,
				OutputFile: outFile,
				IsContents: true,
			})
		default:
			re := regexp.MustCompile(`^(\d+)-(.*?)\.md$`)
			match := re.FindStringSubmatch(filename)
			if len(match) > 2 {
				chapterNum++
				subChapter = 0
				number = fmt.Sprintf("%d.", chapterNum)
				outFile = fmt.Sprintf("%02d.00-%s.html", chapterNum, match[2])
				cat = conf.Categories[chapterNum]
			} else {
				subChapter++
				number = fmt.Sprintf("%d.%d.", chapterNum, subChapter)
				outFile = strings.TrimSuffix(filename, ".md") + ".html"
			}
			chapters = append(chapters, Chapter{
				ID:         strings.TrimSuffix(outFile, ".html"),
				Number:     number,
				Title:      title,
				InputFile:  file,
				OutputFile: outFile,
				Category:   cat,
			})
		}
	}

	// 输出目录
	outDir := conf.OutputDir
	if outputDirOverride != "" {
		outDir = outputDirOverride
	}
	if outDir == "" {
		outDir = "output.html"
	}
	// 如果配置是相对路径，且没有 override (或者 override 也是相对路径)，则相对于 rootDir
	// 注意：如果 override 是 "/tmp/..." 它是绝对路径，IsAbs=true，不会走 Join
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(rootDir, outDir)
	}

	os.MkdirAll(outDir, 0755)
	os.MkdirAll(filepath.Join(outDir, "assets", "css"), 0755)
	os.MkdirAll(filepath.Join(outDir, "assets", "img"), 0755)

	// 复制 CSS (从 embedded templates)
	cssContent, err := templates.Assets.ReadFile("main.css")
	if err != nil {
		fmt.Println("Warning: Could not read embedded CSS (fallback to local if exists)")
	}
	// 尝试优先使用本地 CSS 如果存在
	localCSSPath := filepath.Join(rootDir, "assets", "css", "main.css")
	if localCSS, err := os.ReadFile(localCSSPath); err == nil {
		cssContent = localCSS
	}

	if len(cssContent) > 0 {
		os.WriteFile(filepath.Join(outDir, "assets", "css", "main.css"), cssContent, 0644)
	}

	for i, ch := range chapters {
		mdContent, _ := os.ReadFile(ch.InputFile)

		var htmlContent string
		if ch.IsContents {
			htmlContent = generateTOC(chapters)
		} else {
			htmlContent = markdownToBookHTML(string(mdContent), ch.Number, ch.IsFront)
		}

		prev := ""
		if i > 0 {
			prev = chapters[i-1].OutputFile
		}
		next := ""
		if i < len(chapters)-1 {
			next = chapters[i+1].OutputFile
		}

		pageHTML := buildFullPage(ch, htmlContent, prev, next, chapters)
		os.WriteFile(filepath.Join(outDir, ch.OutputFile), []byte(pageHTML), 0644)
		if ch.IsFront {
			os.WriteFile(filepath.Join(outDir, "index.html"), []byte(pageHTML), 0644)
		}
	}

	fmt.Println("✨ 成功生成电子书到", outDir)
	return nil
}

func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Untitled"
}

func slugify(s string) string {
	s = regexp.MustCompile(`^([第\d\.\s章节：]+)`).ReplaceAllString(s, "")
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// 只保留字母数字和中文
	s = regexp.MustCompile(`[^a-z0-9\p{Han}-]`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`-+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func generateTOC(chapters []Chapter) string {
	var buf bytes.Buffer
	buf.WriteString("<h1 id=\"contents\">目录</h1>\n\n<nav epub:type=\"toc\">\n<ol>\n")

	for _, ch := range chapters {
		if ch.IsContents || ch.IsFront {
			continue
		}

		class := ""
		// If it's a sub-section (e.g., 1.1.), apply indent based on file structure (future proofing)
		if strings.Count(ch.Number, ".") > 1 {
			class = ` class="indent"`
		}

		// Simplify title display
		title := ch.Title
		title = regexp.MustCompile(`^第\s*\d+\s*章[：:]\s*`).ReplaceAllString(title, "")

		// Remove trailing dot from number for display if present
		displayNumber := strings.TrimSuffix(ch.Number, ".")

		// Output format: 1. Introduction
		buf.WriteString(fmt.Sprintf("<li%s><a href=\"%s\">%s. %s</a></li>\n", class, ch.OutputFile, displayNumber, title))

		// Scan for sub-sections (H2) in the markdown file
		content, err := os.ReadFile(ch.InputFile)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "## ") {
					h2Title := strings.TrimSpace(strings.TrimPrefix(line, "## "))
					id := slugify(h2Title)
					buf.WriteString(fmt.Sprintf("<li class=\"indent\"><a href=\"%s#%s\">%s</a></li>\n", ch.OutputFile, id, h2Title))
				}
			}
		}
	}
	buf.WriteString("</ol>\n</nav>\n")
	return buf.String()
}

func buildFullPage(ch Chapter, content, prev, next string, chapters []Chapter) string {
	// Breadcrumbs
	breadcrumb := fmt.Sprintf(`<a href="00.00-front-matter.html">%s</a>`, conf.Title)
	if !ch.IsFront {
		if ch.Category != "" {
			breadcrumb += fmt.Sprintf(` <span class="crumbs">&rsaquo; %s</span>`, ch.Category)
		}
		if !ch.IsContents {
			breadcrumb += fmt.Sprintf(` <span class="crumbs">&rsaquo; %s</span>`, ch.Title)
		} else {
			breadcrumb += ` <span class="crumbs">&rsaquo; 目录</span>`
		}
	}

	// Navigation
	prevLink := `<span class="disabled">上一章</span>`
	prevJS := ""
	if prev != "" {
		prevLink = fmt.Sprintf(`<a href="%s">上一章</a>`, prev)
		prevJS = fmt.Sprintf(`window.location.href = "%s";`, prev)
	}
	nextLink := `<span class="disabled">下一章</span>`
	nextJS := ""
	if next != "" {
		nextLink = fmt.Sprintf(`<a href="%s">下一章</a>`, next)
		nextJS = fmt.Sprintf(`window.location.href = "%s";`, next)
	}

	// Chapter indicator
	chapterDiv := ""
	if ch.Number != "" {
		chapterDiv = fmt.Sprintf(`<div class="chapter">第 %s 章</div>`, strings.TrimSuffix(ch.Number, "."))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
	<head>
		<meta charset="utf-8">
		<meta http-equiv="x-ua-compatible" content="ie=edge">
		<meta name="author" content="%s">
		<meta name="copyright" content="%s">
		<title>%s &mdash; %s</title>
		<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
		<link rel="stylesheet" type="text/css" href="assets/css/main.css">
		<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/intellij-light.min.css">
		<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/languages/go.min.js"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/languages/bash.min.js"></script>
		<script>hljs.highlightAll();</script>
		<script type="module">
			import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
			mermaid.initialize({ startOnLoad: true });
		</script>
	</head>
	<body>
		<header>
			<div class="wrapper">
				<div>
					%s
				</div>
				<div>
					&lsaquo; %s
					&middot; <a href="00.01-contents.html">目录</a> &middot;
					%s &rsaquo;
				</div>
			</div>
		</header>
		<main class="wrapper text">
			%s
			%s
		</main>
		<footer>
			<div class="wrapper">
				<div>
					&lsaquo; %s
				</div>
				<div>
					<a href="00.01-contents.html">目录</a>
				</div>
				<div>
					%s &rsaquo;
				</div>
			</div>
		</footer>
		<script>
			document.onkeydown = function(evt) {
				evt = evt || window.event;
				switch (evt.keyCode) {
					case 37:
						%s
						break;
					case 39:
						%s
						break;
				}
			};

			// Copy button functionality
			document.querySelectorAll('figure.code, figure.bash').forEach(container => {
				const button = document.createElement('button');
				button.className = 'copy-button';
				button.title = 'Copy to clipboard';
				button.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';
				container.appendChild(button);

				button.addEventListener('click', () => {
					const code = container.querySelector('pre').innerText;
					navigator.clipboard.writeText(code).then(() => {
						button.classList.add('copied');
						button.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>';
						setTimeout(() => {
							button.classList.remove('copied');
							button.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>';
						}, 2000);
					}).catch(err => {
						console.error('Failed to copy: ', err);
					});
				});
			});
		</script>
	</body>
</html>
`, conf.Author, conf.Copyright, ch.Title, conf.Title, breadcrumb, prevLink, nextLink, chapterDiv, content, prevLink, nextLink, prevJS, nextJS)
}

func markdownToBookHTML(md string, chapterNum string, isFront bool) string {
	var buf bytes.Buffer

	lines := strings.Split(md, "\n")
	inCodeBlock := false
	inTable := false

	// List tracking
	inList := false
	inLI := false
	currentListTag := "" // "ul" or "ol"

	var codeFileName string
	var codeLang string
	_ = codeLang // avoid unused error

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// 1. 代码块处理
		if strings.HasPrefix(line, "```") {
			// Tables MUST close if we start a code block
			if inTable {
				buf.WriteString("</tbody>\n</table>\n")
				inTable = false
			}

			if inCodeBlock {
				if codeLang == "mermaid" {
					buf.WriteString("</div>\n")
				} else {
					buf.WriteString("</code></pre>\n</figure>\n")
				}
				inCodeBlock = false
				codeFileName = ""
				codeLang = ""
			} else {
				lang := strings.TrimPrefix(line, "```")
				if lang == "" {
					lang = "text"
				}
				codeLang = lang

				if lang == "mermaid" {
					buf.WriteString("<div class=\"mermaid\">\n")
					inCodeBlock = true
					continue
				}

				// 检查下一行是否是文件名注释
				if i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if strings.HasPrefix(nextLine, "// ") || strings.HasPrefix(nextLine, "# ") {
						parts := strings.Fields(nextLine)
						if len(parts) >= 2 && (strings.Contains(parts[1], ".") || strings.Contains(parts[1], "/")) {
							codeFileName = parts[1]
							i++
						}
					}
				}

				buf.WriteString(fmt.Sprintf("<figure class=\"code %s\">\n", lang))
				if codeFileName != "" {
					buf.WriteString(fmt.Sprintf("<figcaption>File: %s</figcaption>\n", codeFileName))
				}
				buf.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">", lang))
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			if codeLang == "mermaid" {
				buf.WriteString(line + "\n")
			} else {
				buf.WriteString(escapeHTML(line) + "\n")
			}
			continue
		}

		// Generic List Termination Check:
		// If we are in a list, and encounter a non-empty line that is NOT indented and NOT a list item,
		// we should close the list. This handles Paragraphs, Headers, Blockquotes, HRs, etc. naturally.
		if inList {
			indent := 0
			for _, r := range line {
				if r == ' ' {
					indent++
				} else if r == '\t' {
					indent += 4
				} else {
					break
				}
			}

			isListItem := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || regexp.MustCompile(`^\d+\. `).MatchString(trimmed)

			// Threshold 2 spaces is standard-ish.
			// If not indented (< 2) and not a list item, close the list.
			if indent < 2 && !isListItem {
				if inLI {
					buf.WriteString("</li>\n")
					inLI = false
				}
				buf.WriteString(fmt.Sprintf("</%s>\n", currentListTag))
				inList = false
				currentListTag = ""
			}
		}

		// 列表闭合检查 (Only Headers, HRs, and Comments close a list once it's started)
		isUL := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ")
		isOL := regexp.MustCompile(`^\d+\. `).MatchString(trimmed)
		isHeader := strings.HasPrefix(trimmed, "#")
		isHR := (trimmed == "---" || trimmed == "***")
		isComment := strings.HasPrefix(trimmed, "<!--")

		if inList && (isHeader || isHR || isComment) {
			if inLI {
				buf.WriteString("</li>\n")
				inLI = false
			}
			buf.WriteString(fmt.Sprintf("</%s>\n", currentListTag))
			inList = false
			currentListTag = ""
		}

		if isComment {
			continue
		}

		// 2. 表格处理
		if strings.HasPrefix(trimmed, "|") && strings.Contains(line, "|") {
			if !inTable {
				buf.WriteString("<table>\n<thead>\n" + renderTableRow(line, "th") + "</thead>\n<tbody>\n")
				inTable = true
				if i+1 < len(lines) && strings.Contains(lines[i+1], "---") {
					i++
				}
			} else {
				buf.WriteString(renderTableRow(line, "td"))
			}
			continue
		} else if inTable {
			buf.WriteString("</tbody>\n</table>\n")
			inTable = false
		}

		// 3. 引用块 -> Aside
		if strings.HasPrefix(line, "> ") {
			var quoteLines []string
			for ; i < len(lines) && strings.HasPrefix(lines[i], "> "); i++ {
				quoteLines = append(quoteLines, strings.TrimPrefix(lines[i], "> "))
			}
			i--

			fullContent := strings.Join(quoteLines, "\n")
			class := "note"
			label := "Note:"
			if strings.Contains(fullContent, "💡") || strings.Contains(fullContent, "提示") {
				class = "hint"
				label = "Hint:"
			} else if strings.Contains(fullContent, "⚠️") || strings.Contains(fullContent, "注意") || strings.Contains(fullContent, "警告") || strings.Contains(fullContent, "重要") {
				class = "important"
				label = "Important:"
			}
			fullContent = regexp.MustCompile(`[💡⚠️❌✅]`).ReplaceAllString(fullContent, "")
			fullContent = strings.TrimSpace(fullContent)

			htmlContent := ""
			for _, qline := range strings.Split(fullContent, "\n") {
				qline = strings.TrimSpace(qline)
				if qline != "" {
					htmlContent += processInline(qline) + "<br>\n"
				}
			}
			htmlContent = strings.TrimSuffix(htmlContent, "<br>\n")

			buf.WriteString(fmt.Sprintf("<aside class=\"%s\"><p>\n<strong>%s</strong> %s\n</p></aside>\n", class, label, htmlContent))
			continue
		}

		// 4. 标题
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimPrefix(line, "# ")
			buf.WriteString(fmt.Sprintf("<h1 id=\"%s\">%s</h1>\n\n", slugify(title), processInline(title)))
			continue
		}
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimPrefix(line, "## ")
			buf.WriteString(fmt.Sprintf("<h2 id=\"%s\">%s</h2>\n\n", slugify(title), processInline(title)))
			continue
		}
		if strings.HasPrefix(line, "### ") {
			title := strings.TrimPrefix(line, "### ")
			buf.WriteString(fmt.Sprintf("<h3 id=\"%s\">%s</h3>\n\n", slugify(title), processInline(title)))
			continue
		}
		if strings.HasPrefix(line, "#### ") {
			title := strings.TrimPrefix(line, "#### ")
			buf.WriteString(fmt.Sprintf("<h4 id=\"%s\">%s</h4>\n\n", slugify(title), processInline(title)))
			continue
		}

		// 5. 水平线
		if trimmed == "---" || trimmed == "***" {
			buf.WriteString("<hr />\n\n")
			continue
		}

		// 6. 列表处理
		if isUL || isOL {
			targetTag := "ul"
			if isOL {
				targetTag = "ol"
			}

			if inList && currentListTag != targetTag {
				if inLI {
					buf.WriteString("</li>\n")
				}
				buf.WriteString(fmt.Sprintf("</%s>\n", currentListTag))
				inList = false
				inLI = false
			}

			if !inList {
				buf.WriteString(fmt.Sprintf("<%s>\n", targetTag))
				inList = true
				currentListTag = targetTag
			} else if inLI {
				buf.WriteString("</li>\n")
			}

			var content string
			if isUL {
				content = strings.TrimPrefix(strings.TrimPrefix(trimmed, "- "), "* ")
			} else {
				content = regexp.MustCompile(`^\d+\. `).ReplaceAllString(trimmed, "")
			}
			buf.WriteString(fmt.Sprintf("<li><p>%s</p>", processInline(content)))
			inLI = true
			continue
		}

		// 8. 普通段落
		if trimmed != "" {
			buf.WriteString(fmt.Sprintf("<p>%s</p>\n\n", processInline(line)))
		}
	}

	if inLI {
		buf.WriteString("</li>\n")
	}
	if inList {
		buf.WriteString(fmt.Sprintf("</%s>\n", currentListTag))
	}
	if inTable {
		buf.WriteString("</tbody>\n</table>\n")
	}

	return buf.String()
}

func renderTableRow(row string, tag string) string {
	parts := strings.Split(strings.Trim(row, "|"), "|")
	var buf bytes.Buffer
	buf.WriteString("<tr>\n")
	for _, p := range parts {
		buf.WriteString(fmt.Sprintf("<%s>%s</%s>\n", tag, processInline(strings.TrimSpace(p)), tag))
	}
	buf.WriteString("</tr>\n")
	return buf.String()
}

func processInline(text string) string {
	// 图片
	text = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`).ReplaceAllString(text, `<figure class="img"><img src="$2" alt="$1"></figure>`)
	// 粗体
	text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(text, "<strong>$1</strong>")
	// 斜体
	text = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(text, "<em>$1</em>")
	// 行内代码
	text = regexp.MustCompile("`([^`]+)`").ReplaceAllString(text, "<code>$1</code>")
	// 链接
	text = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`).ReplaceAllString(text, `<a href="$2">$1</a>`)
	return text
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
