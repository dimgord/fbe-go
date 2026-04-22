// Package html exports a doc.FictionBook as a standalone HTML document.
//
// The output is a single self-contained .html file with embedded CSS and
// embedded binary images (base64 data URLs). Walks the FictionBook struct
// directly instead of transforming XML — simpler than libxslt and keeps the
// CLI pure-Go.
//
// This replaces the COM-plugin / XSLT approach from the original FBE
// (FBE/ExportHTML/html.xsl, 493 lines).
package html

import (
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/dimgord/fbe-go/internal/fb2/doc"
)

// Export writes fb as HTML to w.
func Export(w io.Writer, fb *doc.FictionBook) error {
	e := &exporter{w: w, binaries: map[string]*doc.Binary{}}
	for i := range fb.Binaries {
		e.binaries[fb.Binaries[i].ID] = &fb.Binaries[i]
	}

	e.writeHeader(fb)
	e.writeDescription(&fb.Description)
	for i := range fb.Bodies {
		e.writeBody(&fb.Bodies[i])
	}
	e.writeFooter()
	return e.err
}

type exporter struct {
	w        io.Writer
	err      error
	binaries map[string]*doc.Binary
}

func (e *exporter) write(s string) {
	if e.err != nil {
		return
	}
	_, e.err = io.WriteString(e.w, s)
}

func (e *exporter) writef(format string, args ...any) {
	if e.err != nil {
		return
	}
	_, e.err = fmt.Fprintf(e.w, format, args...)
}

func (e *exporter) writeHeader(fb *doc.FictionBook) {
	var titleStr, langStr string
	if ti := fb.Description.TitleInfo; ti != nil {
		titleStr = ti.BookTitle
		langStr = ti.Lang
	}
	title := html.EscapeString(titleStr)
	e.write(`<!DOCTYPE html>
<html lang="`)
	e.write(html.EscapeString(langStr))
	e.write(`">
<head>
<meta charset="utf-8">
<title>`)
	e.write(title)
	e.write(`</title>
<style>` + exportCSS + `</style>
</head>
<body>
`)
}

func (e *exporter) writeFooter() {
	e.write("</body>\n</html>\n")
}

func (e *exporter) writeDescription(d *doc.Description) {
	ti := d.TitleInfo
	if ti == nil {
		return
	}
	e.write(`<header class="book-meta">` + "\n")
	if cp := ti.Coverpage; cp != nil && len(cp.Images) > 0 {
		src := e.resolveBinaryHref(cp.Images[0].Href)
		if src != "" {
			e.writef("  <img class=\"cover\" src=\"%s\" alt=\"%s\">\n",
				html.EscapeString(src),
				html.EscapeString(ti.BookTitle))
		}
	}
	if ti.BookTitle != "" {
		e.writef("  <h1 class=\"book-title\">%s</h1>\n", html.EscapeString(ti.BookTitle))
	}
	if len(ti.Authors) > 0 {
		e.write(`  <p class="authors">`)
		for i, a := range ti.Authors {
			if i > 0 {
				e.write(", ")
			}
			e.write(html.EscapeString(authorFullName(a)))
		}
		e.write("</p>\n")
	}
	if ti.Annotation != nil {
		e.write(`  <section class="annotation">` + "\n")
		for _, b := range ti.Annotation.Children {
			e.writeBlock(b, "  ")
		}
		e.write("  </section>\n")
	}
	e.write("</header>\n")
}

func authorFullName(a doc.Author) string {
	parts := []string{}
	if a.FirstName != "" {
		parts = append(parts, a.FirstName)
	}
	if a.MiddleName != "" {
		parts = append(parts, a.MiddleName)
	}
	if a.LastName != "" {
		parts = append(parts, a.LastName)
	}
	if len(parts) == 0 && a.Nickname != "" {
		return a.Nickname
	}
	return strings.Join(parts, " ")
}

func (e *exporter) writeBody(b *doc.Body) {
	e.write(`<div class="body"`)
	if b.Name != "" {
		e.writef(` data-name="%s"`, html.EscapeString(b.Name))
	}
	e.write(">\n")
	if b.Title != nil {
		e.writeTitle(b.Title, "h2")
	}
	for i := range b.Epigraph {
		e.writeEpigraph(&b.Epigraph[i])
	}
	if b.Image != nil {
		e.writeImage(b.Image, true)
	}
	for i := range b.Sections {
		e.writeSection(&b.Sections[i], 3)
	}
	e.write("</div>\n")
}

func (e *exporter) writeSection(s *doc.Section, headingLevel int) {
	e.write(`<section class="section"`)
	if s.ID != "" {
		e.writef(` id="%s"`, html.EscapeString(s.ID))
	}
	e.write(">\n")
	if s.Title != nil {
		tag := "h" + fmt.Sprint(clamp(headingLevel, 1, 6))
		e.writeTitle(s.Title, tag)
	}
	for i := range s.Epigraph {
		e.writeEpigraph(&s.Epigraph[i])
	}
	if s.Image != nil {
		e.writeImage(s.Image, true)
	}
	if s.Annotation != nil {
		e.write(`<aside class="annotation">` + "\n")
		for _, b := range s.Annotation.Children {
			e.writeBlock(b, "")
		}
		e.write("</aside>\n")
	}
	if len(s.Sections) > 0 {
		for i := range s.Sections {
			e.writeSection(&s.Sections[i], headingLevel+1)
		}
	} else {
		for _, b := range s.Blocks {
			e.writeBlock(b, "")
		}
	}
	e.write("</section>\n")
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (e *exporter) writeTitle(t *doc.Title, tag string) {
	e.writef("<%s class=\"title\">", tag)
	for i, b := range t.Children {
		if i > 0 {
			e.write("<br>")
		}
		if b.Paragraph != nil {
			e.writeInlines(b.Paragraph.Children)
		}
	}
	e.writef("</%s>\n", tag)
}

func (e *exporter) writeEpigraph(ep *doc.Epigraph) {
	e.write(`<blockquote class="epigraph">` + "\n")
	for _, b := range ep.Children {
		e.writeBlock(b, "")
	}
	for _, p := range ep.TextAuthor {
		e.write(`<p class="text-author">`)
		e.writeInlines(p.Children)
		e.write("</p>\n")
	}
	e.write("</blockquote>\n")
}

func (e *exporter) writeBlock(b doc.Block, _ string) {
	switch {
	case b.Paragraph != nil:
		e.write("<p>")
		e.writeInlines(b.Paragraph.Children)
		e.write("</p>\n")
	case b.EmptyLine != nil:
		e.write(`<p class="empty-line"></p>` + "\n")
	case b.Subtitle != nil:
		e.write(`<p class="subtitle">`)
		e.writeInlines(b.Subtitle.Children)
		e.write("</p>\n")
	case b.Poem != nil:
		e.writePoem(b.Poem)
	case b.Cite != nil:
		e.writeCite(b.Cite)
	case b.Table != nil:
		e.writeTable(b.Table)
	case b.Image != nil:
		e.writeImage(b.Image, true)
	case b.Raw != nil:
		// Best-effort: wrap raw name + text content in a <div>.
		e.writef(`<div data-unknown="%s">`, html.EscapeString(b.Raw.XMLName.Local))
		for _, item := range b.Raw.Items {
			if item.Text != "" {
				e.write(html.EscapeString(item.Text))
			}
		}
		e.write("</div>\n")
	}
}

func (e *exporter) writePoem(p *doc.Poem) {
	e.write(`<div class="poem">` + "\n")
	if p.Title != nil {
		e.writeTitle(p.Title, "h3")
	}
	for i := range p.Epigraph {
		e.writeEpigraph(&p.Epigraph[i])
	}
	for i := range p.Stanzas {
		s := &p.Stanzas[i]
		e.write(`<div class="stanza">` + "\n")
		if s.Title != nil {
			e.writeTitle(s.Title, "h4")
		}
		if s.Subtitle != nil {
			e.write(`<p class="subtitle">`)
			e.writeInlines(s.Subtitle.Children)
			e.write("</p>\n")
		}
		for _, v := range s.Verses {
			e.write(`<p class="v">`)
			e.writeInlines(v.Children)
			e.write("</p>\n")
		}
		e.write("</div>\n")
	}
	for _, ta := range p.TextAuthor {
		e.write(`<p class="text-author">`)
		e.writeInlines(ta.Children)
		e.write("</p>\n")
	}
	e.write("</div>\n")
}

func (e *exporter) writeCite(c *doc.Cite) {
	e.write(`<blockquote class="cite">` + "\n")
	for _, b := range c.Children {
		e.writeBlock(b, "")
	}
	for _, p := range c.TextAuthor {
		e.write(`<p class="text-author">`)
		e.writeInlines(p.Children)
		e.write("</p>\n")
	}
	e.write("</blockquote>\n")
}

func (e *exporter) writeTable(t *doc.Table) {
	e.write(`<table>` + "\n")
	for _, r := range t.Rows {
		e.write("<tr>")
		for _, c := range r.Cells {
			tag := "td"
			if c.XMLName.Local == "th" {
				tag = "th"
			}
			e.writef("<%s", tag)
			if c.ColSpan != "" && c.ColSpan != "1" {
				e.writef(` colspan="%s"`, html.EscapeString(c.ColSpan))
			}
			if c.RowSpan != "" && c.RowSpan != "1" {
				e.writef(` rowspan="%s"`, html.EscapeString(c.RowSpan))
			}
			if c.Align != "" {
				e.writef(` style="text-align:%s"`, html.EscapeString(c.Align))
			}
			e.write(">")
			e.writeInlines(c.Children)
			e.writef("</%s>", tag)
		}
		e.write("</tr>\n")
	}
	e.write("</table>\n")
}

func (e *exporter) writeImage(img *doc.Image, block bool) {
	src := e.resolveBinaryHref(img.Href)
	if src == "" {
		return
	}
	alt := img.Alt
	if alt == "" {
		alt = img.Title
	}
	if block {
		e.writef(`<figure><img src="%s" alt="%s"`, html.EscapeString(src), html.EscapeString(alt))
	} else {
		e.writef(`<img class="inline" src="%s" alt="%s"`, html.EscapeString(src), html.EscapeString(alt))
	}
	if img.Title != "" {
		e.writef(` title="%s"`, html.EscapeString(img.Title))
	}
	e.write(">")
	if block {
		if img.Title != "" {
			e.writef("<figcaption>%s</figcaption>", html.EscapeString(img.Title))
		}
		e.write("</figure>\n")
	}
}

// resolveBinaryHref returns a data: URL for hrefs that reference an embedded
// binary (#id). Returns the href unchanged for external URLs.
func (e *exporter) resolveBinaryHref(href string) string {
	if !strings.HasPrefix(href, "#") {
		return href
	}
	id := strings.TrimPrefix(href, "#")
	bin, ok := e.binaries[id]
	if !ok {
		return ""
	}
	ct := bin.ContentType
	if ct == "" {
		ct = "image/jpeg"
	}
	// Validate base64 cheaply; if it fails, skip the image.
	if _, err := base64.StdEncoding.DecodeString(strings.TrimSpace(bin.Data)); err != nil {
		return ""
	}
	return "data:" + ct + ";base64," + strings.TrimSpace(bin.Data)
}

func (e *exporter) writeInlines(items []doc.Inline) {
	for _, i := range items {
		e.writeInline(i)
	}
}

func (e *exporter) writeInline(i doc.Inline) {
	if i.Text != "" {
		e.write(html.EscapeString(i.Text))
	}
	switch {
	case i.Strong != nil:
		e.write("<strong>")
		e.writeInlines(i.Strong.Children)
		e.write("</strong>")
	case i.Emphasis != nil:
		e.write("<em>")
		e.writeInlines(i.Emphasis.Children)
		e.write("</em>")
	case i.Strikethrough != nil:
		e.write("<s>")
		e.writeInlines(i.Strikethrough.Children)
		e.write("</s>")
	case i.Sub != nil:
		e.write("<sub>")
		e.writeInlines(i.Sub.Children)
		e.write("</sub>")
	case i.Sup != nil:
		e.write("<sup>")
		e.writeInlines(i.Sup.Children)
		e.write("</sup>")
	case i.Code != nil:
		e.write("<code>")
		e.writeInlines(i.Code.Children)
		e.write("</code>")
	case i.Style != nil:
		e.writef(`<span class="style-%s">`, html.EscapeString(i.Style.Name))
		e.writeInlines(i.Style.Children)
		e.write("</span>")
	case i.A != nil:
		e.writef(`<a href="%s"`, html.EscapeString(i.A.Href))
		if i.A.Type != "" {
			e.writef(` class="link-%s"`, html.EscapeString(i.A.Type))
		}
		e.write(">")
		e.writeInlines(i.A.Children)
		e.write("</a>")
	case i.Image != nil:
		e.writeImage(i.Image, false)
	}
}

// exportCSS is a minimal book-style CSS embedded into every generated HTML.
const exportCSS = `
body { max-width: 42rem; margin: 2rem auto; padding: 0 1rem;
  font-family: Georgia, "Times New Roman", serif; line-height: 1.55; color: #222; }
.book-meta { text-align: center; margin-bottom: 3rem; border-bottom: 1px solid #ddd; padding-bottom: 2rem; }
.book-meta .cover { max-width: 16rem; height: auto; margin-bottom: 1rem; }
.book-title { font-size: 2rem; margin: 0.3rem 0; }
.authors { color: #666; font-style: italic; }
.annotation { text-align: left; margin-top: 1rem; font-size: 0.95em; }
.body { margin-bottom: 2rem; }
.section { margin: 1.5rem 0; }
.title { text-align: center; }
.epigraph, .cite { margin: 1rem 0 1rem 3em; font-style: italic; color: #555; border-left: 2px solid #ccc; padding-left: 1em; }
.text-author { text-align: right; margin-top: 0.3rem; }
.subtitle { font-weight: 600; font-size: 1.1em; margin-top: 1em; }
.empty-line { height: 1em; margin: 0; }
.poem { margin: 1.5rem 0 1.5rem 2rem; }
.stanza { margin-bottom: 1rem; }
.v { margin: 0; }
table { border-collapse: collapse; margin: 1rem 0; }
th, td { border: 1px solid #d0d0c0; padding: 0.3em 0.7em; }
th { background: #f0f0ea; }
code { font-family: "SF Mono", Menlo, monospace; background: #f5f5ef; padding: 0.1em 0.3em; border-radius: 3px; }
figure { text-align: center; margin: 1rem 0; }
figure img { max-width: 100%; height: auto; }
figcaption { font-size: 0.9em; color: #666; margin-top: 0.3rem; }
img.inline { max-height: 1.2em; vertical-align: middle; }
`
