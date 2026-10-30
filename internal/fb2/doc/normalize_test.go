package doc

import "testing"

func TestNormalizeDropsEmptyPlaceholders(t *testing.T) {
	fb := &FictionBook{}
	fb.Description.TitleInfo = &TitleInfo{Coverpage: &Coverpage{}, Annotation: &Annotation{}}
	fb.Description.SrcTitleInfo = &TitleInfo{Annotation: &Annotation{Children: []Block{}}}
	fb.Description.PublishInfo = &PublishInfo{Sequences: []Sequence{}}
	fb.Normalize()
	d := fb.Description
	if d.TitleInfo.Coverpage != nil || d.TitleInfo.Annotation != nil ||
		d.SrcTitleInfo.Annotation != nil || d.PublishInfo != nil {
		t.Fatalf("placeholders left: %+v", d)
	}
}

func TestNormalizeKeepsContent(t *testing.T) {
	fb := &FictionBook{}
	fb.Description.TitleInfo = &TitleInfo{
		Coverpage:  &Coverpage{Images: []Image{{}}},
		Annotation: &Annotation{ID: "a1"},
	}
	fb.Description.PublishInfo = &PublishInfo{Year: "1999"}
	fb.Normalize()
	d := fb.Description
	if d.TitleInfo.Coverpage == nil || d.TitleInfo.Annotation == nil || d.PublishInfo == nil {
		t.Fatalf("content dropped: %+v", d)
	}
}
