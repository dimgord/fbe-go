package doc

// Normalize drops sub-elements that would round-trip as empty, XSD-invalid placeholders.
// An empty <coverpage> fails the schema, which requires at least one <image>.
func (fb *FictionBook) Normalize() {
	for _, ti := range []*TitleInfo{fb.Description.TitleInfo, fb.Description.SrcTitleInfo} {
		if ti != nil && ti.Coverpage != nil && len(ti.Coverpage.Images) == 0 {
			ti.Coverpage = nil
		}
	}
}
