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
// The XMLName binds the FB2 namespace so the writer emits xmlns at the root.
type FictionBook struct {
	XMLName     xml.Name     `xml:"http://www.gribuser.ru/xml/fictionbook/2.0 FictionBook"`
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
//
// TitleInfo is a pointer with `,omitempty` (rather than a value type) so that
// a document whose source lacks `<title-info>` round-trips as "absent" rather
// than materializing a ghost element with empty `<book-title>`/`<lang>` on the
// writer side. Callers must nil-check before dereferencing.
type Description struct {
	TitleInfo    *TitleInfo    `xml:"title-info,omitempty"`
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
// Exactly one of the pointer fields is non-nil; custom Marshal/Unmarshal
// dispatch on element name.
//
// Raw captures any element we don't have a typed representation for (custom
// FB2 extensions, future-version elements) so the round-trip is lossless.
type Block struct {
	Paragraph *Paragraph
	Poem      *Poem
	Subtitle  *Paragraph
	Cite      *Cite
	EmptyLine *EmptyLine
	Table     *Table
	Image     *Image
	Raw       *RawElement
}

// RawElement preserves an arbitrary XML element verbatim. Used as a fallback
// for unknown elements so that parse → write does not drop content.
type RawElement struct {
	XMLName xml.Name
	Attrs   []xml.Attr
	Items   []RawItem
}

// RawItem is either text content or a nested RawElement.
type RawItem struct {
	Text string
	Elem *RawElement
}

// UnmarshalXML captures element name, attributes, and all child tokens.
func (r *RawElement) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	r.XMLName = start.Name
	r.Attrs = append([]xml.Attr(nil), start.Attr...)
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.CharData:
			if s := string(t); s != "" {
				r.Items = append(r.Items, RawItem{Text: s})
			}
		case xml.StartElement:
			var sub RawElement
			if err := sub.UnmarshalXML(d, t); err != nil {
				return err
			}
			r.Items = append(r.Items, RawItem{Elem: &sub})
		case xml.EndElement:
			return nil
		}
	}
}

// MarshalXML re-emits the element with its original name, attributes, and
// children (text + nested raw elements).
func (r RawElement) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	// Use only the local name so the ambient xmlns inherited from the root
	// applies; never re-emit xmlns= on unknown elements.
	start := xml.StartElement{
		Name: xml.Name{Local: r.XMLName.Local},
		Attr: r.Attrs,
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, it := range r.Items {
		if it.Text != "" {
			if err := e.EncodeToken(xml.CharData(it.Text)); err != nil {
				return err
			}
		} else if it.Elem != nil {
			if err := it.Elem.MarshalXML(e, xml.StartElement{}); err != nil {
				return err
			}
		}
	}
	return e.EncodeToken(start.End())
}

// UnmarshalXML dispatches on the element name (local part, namespace ignored).
// Unknown elements are preserved verbatim via Raw for lossless round-trip.
func (b *Block) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	switch start.Name.Local {
	case "p":
		b.Paragraph = &Paragraph{}
		return d.DecodeElement(b.Paragraph, &start)
	case "poem":
		b.Poem = &Poem{}
		return d.DecodeElement(b.Poem, &start)
	case "subtitle":
		b.Subtitle = &Paragraph{}
		return d.DecodeElement(b.Subtitle, &start)
	case "cite":
		b.Cite = &Cite{}
		return d.DecodeElement(b.Cite, &start)
	case "empty-line":
		b.EmptyLine = &EmptyLine{}
		return d.DecodeElement(b.EmptyLine, &start)
	case "table":
		b.Table = &Table{}
		return d.DecodeElement(b.Table, &start)
	case "image":
		b.Image = &Image{}
		return d.DecodeElement(b.Image, &start)
	}
	b.Raw = &RawElement{}
	return b.Raw.UnmarshalXML(d, start)
}

// MarshalXML emits the element corresponding to whichever field is populated.
func (b Block) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	switch {
	case b.Paragraph != nil:
		return e.EncodeElement(b.Paragraph, xml.StartElement{Name: xml.Name{Local: "p"}})
	case b.Poem != nil:
		return e.EncodeElement(b.Poem, xml.StartElement{Name: xml.Name{Local: "poem"}})
	case b.Subtitle != nil:
		return e.EncodeElement(b.Subtitle, xml.StartElement{Name: xml.Name{Local: "subtitle"}})
	case b.Cite != nil:
		return e.EncodeElement(b.Cite, xml.StartElement{Name: xml.Name{Local: "cite"}})
	case b.EmptyLine != nil:
		return e.EncodeElement(b.EmptyLine, xml.StartElement{Name: xml.Name{Local: "empty-line"}})
	case b.Table != nil:
		return e.EncodeElement(b.Table, xml.StartElement{Name: xml.Name{Local: "table"}})
	case b.Image != nil:
		return e.EncodeElement(b.Image, xml.StartElement{Name: xml.Name{Local: "image"}})
	case b.Raw != nil:
		return b.Raw.MarshalXML(e, xml.StartElement{})
	}
	return nil
}

// EmptyLine — FB2's explicit blank line.
type EmptyLine struct {
	ID string `xml:"id,attr,omitempty"`
}

// Paragraph = run of inline nodes with optional style/id.
// Children is populated via custom UnmarshalXML so text and elements interleave.
type Paragraph struct {
	ID       string   `xml:"id,attr,omitempty"`
	Style    string   `xml:"style,attr,omitempty"`
	Lang     string   `xml:"lang,attr,omitempty"`
	Children []Inline `xml:"-"`
}

// UnmarshalXML reads mixed content (text + inline elements) into Children.
func (p *Paragraph) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// Attributes first.
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "id":
			p.ID = a.Value
		case "style":
			p.Style = a.Value
		case "lang":
			p.Lang = a.Value
		}
	}
	return unmarshalInlineContent(d, start, &p.Children)
}

// MarshalXML emits attributes and then re-serializes Children as mixed content.
//
// We don't attempt to suppress Go's encoder indent around the mixed-content
// children from here — `xml.Encoder.Indent("", "")` would produce visually
// correct output for this <p>'s children, but it leaves the encoder's
// *internal* depth counter out of sync with the tag stack (writeIndent with
// both strings empty short-circuits before decrementing the counter), which
// then over-indents every following sibling block by the number of toggles.
// The mixed-content whitespace cleanup is done once in `writer.Write` as a
// targeted regex pass — see the comment there for the regex and the set of
// affected container names.
func (p Paragraph) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	addAttrIfSet(&start, "id", p.ID)
	addAttrIfSet(&start, "style", p.Style)
	addAttrIfSet(&start, "lang", p.Lang)
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := marshalInlineContent(e, p.Children); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

// addAttrIfSet appends an attribute to the start element if its value is non-empty.
func addAttrIfSet(start *xml.StartElement, name, value string) {
	if value != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: value})
	}
}

// unmarshalInlineContent reads tokens until the matching end element, treating
// chardata as a text Inline and nested elements as marks/images/links.
func unmarshalInlineContent(d *xml.Decoder, start xml.StartElement, out *[]Inline) error {
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.CharData:
			if s := string(t); s != "" {
				*out = append(*out, Inline{Text: s})
			}
		case xml.StartElement:
			var inl Inline
			switch t.Name.Local {
			case "strong":
				inl.Strong = &Paragraph{}
				if err := d.DecodeElement(inl.Strong, &t); err != nil {
					return err
				}
			case "emphasis":
				inl.Emphasis = &Paragraph{}
				if err := d.DecodeElement(inl.Emphasis, &t); err != nil {
					return err
				}
			case "style":
				inl.Style = &StyleInline{}
				if err := d.DecodeElement(inl.Style, &t); err != nil {
					return err
				}
			case "a":
				inl.A = &Link{}
				if err := d.DecodeElement(inl.A, &t); err != nil {
					return err
				}
			case "strikethrough":
				inl.Strikethrough = &Paragraph{}
				if err := d.DecodeElement(inl.Strikethrough, &t); err != nil {
					return err
				}
			case "sub":
				inl.Sub = &Paragraph{}
				if err := d.DecodeElement(inl.Sub, &t); err != nil {
					return err
				}
			case "sup":
				inl.Sup = &Paragraph{}
				if err := d.DecodeElement(inl.Sup, &t); err != nil {
					return err
				}
			case "code":
				inl.Code = &Paragraph{}
				if err := d.DecodeElement(inl.Code, &t); err != nil {
					return err
				}
			case "image":
				inl.Image = &Image{}
				if err := d.DecodeElement(inl.Image, &t); err != nil {
					return err
				}
			default:
				inl.Raw = &RawElement{}
				if err := inl.Raw.UnmarshalXML(d, t); err != nil {
					return err
				}
			}
			*out = append(*out, inl)
		case xml.EndElement:
			if t.Name == start.Name {
				return nil
			}
		}
	}
}

// marshalInlineContent re-emits the Inline children collected by Paragraph.
func marshalInlineContent(e *xml.Encoder, children []Inline) error {
	for _, in := range children {
		switch {
		case in.Text != "":
			if err := e.EncodeToken(xml.CharData(in.Text)); err != nil {
				return err
			}
		case in.Strong != nil:
			if err := e.EncodeElement(in.Strong, xml.StartElement{Name: xml.Name{Local: "strong"}}); err != nil {
				return err
			}
		case in.Emphasis != nil:
			if err := e.EncodeElement(in.Emphasis, xml.StartElement{Name: xml.Name{Local: "emphasis"}}); err != nil {
				return err
			}
		case in.Style != nil:
			if err := e.EncodeElement(in.Style, xml.StartElement{Name: xml.Name{Local: "style"}}); err != nil {
				return err
			}
		case in.A != nil:
			if err := e.EncodeElement(in.A, xml.StartElement{Name: xml.Name{Local: "a"}}); err != nil {
				return err
			}
		case in.Strikethrough != nil:
			if err := e.EncodeElement(in.Strikethrough, xml.StartElement{Name: xml.Name{Local: "strikethrough"}}); err != nil {
				return err
			}
		case in.Sub != nil:
			if err := e.EncodeElement(in.Sub, xml.StartElement{Name: xml.Name{Local: "sub"}}); err != nil {
				return err
			}
		case in.Sup != nil:
			if err := e.EncodeElement(in.Sup, xml.StartElement{Name: xml.Name{Local: "sup"}}); err != nil {
				return err
			}
		case in.Code != nil:
			if err := e.EncodeElement(in.Code, xml.StartElement{Name: xml.Name{Local: "code"}}); err != nil {
				return err
			}
		case in.Image != nil:
			if err := e.EncodeElement(in.Image, xml.StartElement{Name: xml.Name{Local: "image"}}); err != nil {
				return err
			}
		case in.Raw != nil:
			if err := in.Raw.MarshalXML(e, xml.StartElement{}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Inline — inline content: plain text, marks, images, links. Exactly one field
// is non-zero per Inline (Text alone; or one of the element pointers).
// Raw preserves unknown inline elements verbatim for lossless round-trip.
type Inline struct {
	Text          string
	Strong        *Paragraph
	Emphasis      *Paragraph
	Style         *StyleInline
	A             *Link
	Strikethrough *Paragraph
	Sub           *Paragraph
	Sup           *Paragraph
	Code          *Paragraph
	Image         *Image
	Raw           *RawElement
}

// StyleInline — named inline style (<style name="...">).
type StyleInline struct {
	Name     string   `xml:"name,attr"`
	Children []Inline `xml:"-"`
}

// UnmarshalXML reads the name attribute and mixed inline content.
func (s *StyleInline) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		if a.Name.Local == "name" {
			s.Name = a.Value
		}
	}
	return unmarshalInlineContent(d, start, &s.Children)
}

// MarshalXML re-emits attribute + children.
func (s StyleInline) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	addAttrIfSet(&start, "name", s.Name)
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := marshalInlineContent(e, s.Children); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

// Link = FB2 <a> — note references use type="note".
//
// Href uses `xml:"-"` and a manually-emitted `l:href` attribute literal
// (colon-in-local-name) in MarshalXML. Using `xml:"http://…/xlink href,attr"`
// would trigger Go's encoding/xml namespace handler, which re-declares
// `xmlns:xlink="http://…/xlink"` on every `<a>` element instead of reusing
// the `xmlns:l` prefix declared once on the FictionBook root. The
// unmarshal side accepts any of `l:href`, `xlink:href`, bare `href`, or
// the parser's namespace-resolved variant with Space == NSXLink.
type Link struct {
	Href     string   `xml:"-"`
	Type     string   `xml:"type,attr,omitempty"`
	Children []Inline `xml:"-"`
}

// UnmarshalXML reads href/type attributes and mixed inline content.
func (l *Link) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, a := range start.Attr {
		switch {
		case a.Name.Local == "href" && (a.Name.Space == NSXLink || a.Name.Space == ""):
			l.Href = a.Value
		case a.Name.Local == "type":
			l.Type = a.Value
		}
	}
	return unmarshalInlineContent(d, start, &l.Children)
}

// MarshalXML emits l:href (xlink) and type attribute, plus mixed content.
// The `l:href` attribute name is emitted as a literal local name (colon
// included) so Go's default namespace machinery doesn't kick in and
// re-declare `xmlns:xlink` on every <a>. The FictionBook root (emitted by
// writer.Write) is responsible for declaring `xmlns:l` once.
func (l Link) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if l.Href != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "l:href"}, Value: l.Href})
	}
	addAttrIfSet(&start, "type", l.Type)
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := marshalInlineContent(e, l.Children); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
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
//
// Children uses `xml:"-"` + custom (Un)MarshalXML — same pattern as Paragraph.
// With `xml:",any"` Go's default marshaller would emit `<Children>` as the
// element name (from the field name) and serialize each Inline struct field
// (Text/Strong/…) with its Go name, producing invalid FB2 like
// `<th><Children><Text>…</Text></Children></th>`. Instead we dispatch inline
// content through `marshalInlineContent` so nested marks and text interleave
// directly inside <th>/<td>.
type Cell struct {
	XMLName  xml.Name // local name "th" or "td"
	ID       string   `xml:"id,attr,omitempty"`
	Style    string   `xml:"style,attr,omitempty"`
	ColSpan  string   `xml:"colspan,attr,omitempty"`
	RowSpan  string   `xml:"rowspan,attr,omitempty"`
	Align    string   `xml:"align,attr,omitempty"`
	VAlign   string   `xml:"valign,attr,omitempty"`
	Children []Inline `xml:"-"`
}

// UnmarshalXML reads th/td attributes and mixed inline content.
func (c *Cell) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	c.XMLName = start.Name
	for _, a := range start.Attr {
		switch a.Name.Local {
		case "id":
			c.ID = a.Value
		case "style":
			c.Style = a.Value
		case "colspan":
			c.ColSpan = a.Value
		case "rowspan":
			c.RowSpan = a.Value
		case "align":
			c.Align = a.Value
		case "valign":
			c.VAlign = a.Value
		}
	}
	return unmarshalInlineContent(d, start, &c.Children)
}

// MarshalXML emits <th|td> with attributes and inline mixed content.
//
// Uses only the local name — not c.XMLName in full — so the parent's xmlns
// default namespace applies without each cell re-emitting
// `xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"`. Fall back to the
// incoming start.Name.Local when c.XMLName is zero (shouldn't happen in
// practice since it's populated during unmarshal and by the frontend).
//
// Mixed-content whitespace (text + inline marks inside the cell) is
// normalized by the writer's post-process pass, not here. See the long
// comment on Paragraph.MarshalXML for why toggling encoder indent from a
// nested MarshalXML corrupts the encoder's depth counter.
func (c Cell) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	local := c.XMLName.Local
	if local == "" {
		local = start.Name.Local
	}
	start.Name = xml.Name{Local: local}
	start.Attr = nil
	addAttrIfSet(&start, "id", c.ID)
	addAttrIfSet(&start, "style", c.Style)
	addAttrIfSet(&start, "colspan", c.ColSpan)
	addAttrIfSet(&start, "rowspan", c.RowSpan)
	addAttrIfSet(&start, "align", c.Align)
	addAttrIfSet(&start, "valign", c.VAlign)
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := marshalInlineContent(e, c.Children); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

// Binary holds a base64-encoded binary (typically an image).
type Binary struct {
	ID          string `xml:"id,attr"`
	ContentType string `xml:"content-type,attr"`
	Data        string `xml:",chardata"` // base64
}
