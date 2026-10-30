package doc

// Normalize drops sub-elements that would round-trip as empty placeholders. An empty <coverpage>
// fails the schema, which requires an <image>; empty <publish-info> and <annotation> are valid but
// are noise the description forms create just by being opened, so they must not reach the file.
func (fb *FictionBook) Normalize() {
	d := &fb.Description
	for _, ti := range []*TitleInfo{d.TitleInfo, d.SrcTitleInfo} {
		if ti == nil {
			continue
		}
		if ti.Coverpage != nil && len(ti.Coverpage.Images) == 0 {
			ti.Coverpage = nil
		}
		if a := ti.Annotation; a != nil && a.ID == "" && a.Lang == "" && len(a.Children) == 0 {
			ti.Annotation = nil
		}
	}
	if p := d.PublishInfo; p != nil && p.BookName == "" && p.Publisher == "" && p.City == "" &&
		p.Year == "" && p.ISBN == "" && len(p.Sequences) == 0 {
		d.PublishInfo = nil
	}
}
