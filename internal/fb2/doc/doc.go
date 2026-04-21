// Package doc defines the in-memory representation of a FictionBook 2 document.
//
// The model is schema-faithful to FictionBook.xsd and intentionally separate from
// any editor DOM. The parser builds values of these types; the writer serializes
// them back to canonical FB2 XML. The frontend (ProseMirror) uses its own document
// model and converts to/from these types at the Wails boundary.
package doc

import "encoding/xml"

// Namespaces used in FB2.
const (
	NSFictionBook = "http://www.gribuser.ru/xml/fictionbook/2.0"
	NSXLink       = "http://www.w3.org/1999/xlink"
)

// FictionBook is the root element of an FB2 document.
type FictionBook struct {
	XMLName     xml.Name     `xml:"FictionBook"`
	Stylesheets []Stylesheet `xml:"stylesheet,omitempty"`
	Description Description  `xml:"description"`
	Bodies      []Body       `xml:"body"`
	Binaries    []Binary     `xml:"binary,omitempty"`
}

// Stylesheet carries CSS-like content embedded in the document.
type Stylesheet struct {
	Type    string `xml:"type,attr"`
	Content string `xml:",chardata"`
}

// Description aggregates all descriptive metadata sections.
type Description struct {
	TitleInfo    TitleInfo     `xml:"title-info"`
	SrcTitleInfo *TitleInfo    `xml:"src-title-info,omitempty"`
	DocumentInfo DocumentInfo  `xml:"document-info"`
	PublishInfo  *PublishInfo  `xml:"publish-info,omitempty"`
	CustomInfo   []CustomInfo  `xml:"custom-info,omitempty"`
	Output       []OutputBlock `xml:"output,omitempty"`
}

// TitleInfo and SrcTitleInfo share the same fields in FB2.
type TitleInfo struct {
	Genres     []Genre     `xml:"genre"`
	Authors    []Author    `xml:"author"`
	BookTitle  string      `xml:"book-title"`
	Annotation *Annotation `xml:"annotation,omitempty"`
	Keywords   string      `xml:"keywords,omitempty"`
	Date       *Date       `xml:"date,omitempty"`
	Coverpage  *Coverpage  `xml:"coverpage,omitempty"`
	Lang       string      `xml:"lang"`
	SrcLang    string      `xml:"src-lang,omitempty"`
	Translators []Author   `xml:"translator,omitempty"`
	Sequences  []Sequence  `xml:"sequence,omitempty"`
}

// Genre with optional match percentage.
type Genre struct {
	Value string `xml:",chardata"`
	Match string `xml:"match,attr,omitempty"`
}

// Author describes a person (author / translator / doc-info author).
type Author struct {
	FirstName  string   `xml:"first-name,omitempty"`
	MiddleName string   `xml:"middle-name,omitempty"`
	LastName   string   `xml:"last-name,omitempty"`
	Nickname   string   `xml:"nickname,omitempty"`
	HomePage   []string `xml:"home-page,omitempty"`
	Email      []string `xml:"email,omitempty"`
	ID         string   `xml:"id,omitempty"`
}

// Date has a human-readable body and machine ISO value.
type Date struct {
	Value string `xml:"value,attr,omitempty"`
	Text  string `xml:",chardata"`
}

// Coverpage holds one or more cover images (inline/link).
type Coverpage struct {
	Images []Image `xml:"image"`
}

// Sequence supports recursive nesting (series within series).
type Sequence struct {
	Name     string     `xml:"name,attr"`
	Number   string     `xml:"number,attr,omitempty"`
	Children []Sequence `xml:"sequence,omitempty"`
}

// DocumentInfo carries document-level metadata (FB-specific).
type DocumentInfo struct {
	Authors     []Author `xml:"author"`
	ProgramUsed string   `xml:"program-used,omitempty"`
	Date        Date     `xml:"date"`
	SrcURL      []string `xml:"src-url,omitempty"`
	SrcOCR      string   `xml:"src-ocr,omitempty"`
	ID          string   `xml:"id"`
	Version     string   `xml:"version"`
	History     *History `xml:"history,omitempty"`
	Publishers  []Author `xml:"publisher,omitempty"`
}

// PublishInfo — print/paper book info.
type PublishInfo struct {
	BookName  string     `xml:"book-name,omitempty"`
	Publisher string     `xml:"publisher,omitempty"`
	City      string     `xml:"city,omitempty"`
	Year      string     `xml:"year,omitempty"`
	ISBN      string     `xml:"isbn,omitempty"`
	Sequences []Sequence `xml:"sequence,omitempty"`
}

// CustomInfo is a free-form typed string.
type CustomInfo struct {
	InfoType string `xml:"info-type,attr"`
	Value    string `xml:",chardata"`
}

// OutputBlock corresponds to FB2 <output> — rarely used, kept for fidelity.
type OutputBlock struct {
	Mode          string `xml:"mode,attr"`
	IncludeAll    string `xml:"include-all,attr"`
	Price         string `xml:"price,attr,omitempty"`
	Currency      string `xml:"currency,attr,omitempty"`
	// ...spec has additional attrs; skipping for now
}

// Annotation is a rich-text container used in title-info and sections.
type Annotation struct {
	ID       string    `xml:"id,attr,omitempty"`
	Lang     string    `xml:"lang,attr,omitempty"`
	Children []Block   `xml:",any"`
}

// History — same shape as annotation (paragraphs and friends).
type History struct {
	ID       string  `xml:"id,attr,omitempty"`
	Children []Block `xml:",any"`
}

// Body — main content container. Multiple bodies are allowed (e.g., footnotes).
type Body struct {
	Name     string    `xml:"name,attr,omitempty"`
	Lang     string    `xml:"lang,attr,omitempty"`
	Image    *Image    `xml:"image,omitempty"`
	Title    *Title    `xml:"title,omitempty"`
	Epigraph []Epigraph `xml:"epigraph,omitempty"`
	Sections []Section `xml:"section"`
}

// Section is a recursive structural block.
type Section struct {
	ID       string    `xml:"id,attr,omitempty"`
	Title    *Title    `xml:"title,omitempty"`
	Epigraph []Epigraph `xml:"epigraph,omitempty"`
	Image    *Image    `xml:"image,omitempty"`
	Annotation *Annotation `xml:"annotation,omitempty"`

	// Either nested sections OR inline block content (paragraphs, poems, etc.).
	Sections []Section `xml:"section,omitempty"`
	Blocks   []Block   `xml:",any"`
}

// Title is a simple block container of <p> / <empty-line>.
type Title struct {
	ID       string  `xml:"id,attr,omitempty"`
	Children []Block `xml:",any"`
}

// Epigraph is a section-like block with optional text-author trailer.
type Epigraph struct {
	ID         string      `xml:"id,attr,omitempty"`
	Children   []Block     `xml:",any"`
	TextAuthor []Paragraph `xml:"text-author,omitempty"`
}

// Cite — quoted block with optional text-author trailer.
type Cite struct {
	ID         string      `xml:"id,attr,omitempty"`
	Lang       string      `xml:"lang,attr,omitempty"`
	Children   []Block     `xml:",any"`
	TextAuthor []Paragraph `xml:"text-author,omitempty"`
}

// Poem — stanza container.
type Poem struct {
	ID         string      `xml:"id,attr,omitempty"`
	Lang       string      `xml:"lang,attr,omitempty"`
	Title      *Title      `xml:"title,omitempty"`
	Epigraph   []Epigraph  `xml:"epigraph,omitempty"`
	Stanzas    []Stanza    `xml:"stanza"`
	TextAuthor []Paragraph `xml:"text-author,omitempty"`
	Date       *Date       `xml:"date,omitempty"`
}

// Stanza — group of verse lines with optional title/subtitle.
type Stanza struct {
	ID       string    `xml:"id,attr,omitempty"`
	Title    *Title    `xml:"title,omitempty"`
	Subtitle *Paragraph `xml:"subtitle,omitempty"`
	Verses   []Paragraph `xml:"v"`
}

// Block is the union of block-level nodes inside rich-text containers.
// Only one field is non-zero at a time. Parser/writer dispatch on tag name.
type Block struct {
	XMLName   xml.Name
	Paragraph *Paragraph  `xml:"p,omitempty"`
	Poem      *Poem       `xml:"poem,omitempty"`
	Subtitle  *Paragraph  `xml:"subtitle,omitempty"`
	Cite      *Cite       `xml:"cite,omitempty"`
	EmptyLine *EmptyLine  `xml:"empty-line,omitempty"`
	Table     *Table      `xml:"table,omitempty"`
	Image     *Image      `xml:"image,omitempty"`

	// Raw fallback: any unknown block-level element.
	Raw []byte `xml:",innerxml"`
}

// EmptyLine — FB2's explicit blank line.
type EmptyLine struct {
	ID string `xml:"id,attr,omitempty"`
}

// Paragraph = run of inline nodes with optional style/id.
type Paragraph struct {
	ID       string   `xml:"id,attr,omitempty"`
	Style    string   `xml:"style,attr,omitempty"`
	Lang     string   `xml:"lang,attr,omitempty"`
	Children []Inline `xml:",any"`
}

// Inline — inline content: text, marks, images, links.
type Inline struct {
	XMLName      xml.Name
	Text         string       `xml:",chardata"`
	Strong       *Paragraph   `xml:"strong,omitempty"`
	Emphasis     *Paragraph   `xml:"emphasis,omitempty"`
	Style        *StyleInline `xml:"style,omitempty"`
	A            *Link        `xml:"a,omitempty"`
	Strikethrough *Paragraph  `xml:"strikethrough,omitempty"`
	Sub          *Paragraph   `xml:"sub,omitempty"`
	Sup          *Paragraph   `xml:"sup,omitempty"`
	Code         *Paragraph   `xml:"code,omitempty"`
	Image        *Image       `xml:"image,omitempty"`

	Raw []byte `xml:",innerxml"`
}

// StyleInline — named inline style (<style name="...">).
type StyleInline struct {
	Name     string   `xml:"name,attr"`
	Children []Inline `xml:",any"`
}

// Link = FB2 <a> — note references use type="note".
type Link struct {
	Href     string   `xml:"http://www.w3.org/1999/xlink href,attr"`
	Type     string   `xml:"type,attr,omitempty"`
	Children []Inline `xml:",any"`
}

// Image — block or inline; distinguished by position in the tree.
type Image struct {
	Href  string `xml:"http://www.w3.org/1999/xlink href,attr"`
	Alt   string `xml:"alt,attr,omitempty"`
	Title string `xml:"title,attr,omitempty"`
	ID    string `xml:"id,attr,omitempty"`
}

// Table — FB2 <table> with rows.
type Table struct {
	ID    string `xml:"id,attr,omitempty"`
	Style string `xml:"style,attr,omitempty"`
	Rows  []Row  `xml:"tr"`
}

// Row = <tr>.
type Row struct {
	Align string `xml:"align,attr,omitempty"`
	Cells []Cell `xml:",any"`
}

// Cell = <th> or <td>.
type Cell struct {
	XMLName  xml.Name // local name "th" or "td"
	ID       string   `xml:"id,attr,omitempty"`
	Style    string   `xml:"style,attr,omitempty"`
	ColSpan  string   `xml:"colspan,attr,omitempty"`
	RowSpan  string   `xml:"rowspan,attr,omitempty"`
	Align    string   `xml:"align,attr,omitempty"`
	VAlign   string   `xml:"valign,attr,omitempty"`
	Children []Inline `xml:",any"`
}

// Binary holds a base64-encoded binary (typically an image).
type Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"` // base64
}
