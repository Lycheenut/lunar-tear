package masterdataadmin

import "testing"

func TestEventChapterTitlesCombinesDistinctRelations(t *testing.T) {
	resolver := &titleResolver{
		texts: localizationIndex{
			"en": {
				"quest.event.chapter_title.101": "Easy",
				"quest.event.chapter_title.102": "Normal",
				"quest.event.chapter_title.103": "Normal",
			},
			"ja": {
				"quest.event.chapter_title.101": "初級",
				"quest.event.chapter_title.102": "中級",
				"quest.event.chapter_title.103": "中級",
			},
			"ko": {},
		},
		chapterTextIDs: map[int64]int64{1: 101, 2: 102, 3: 103},
	}
	titles := resolver.eventChapterTitles([]int64{1, 2, 3})
	if got, want := titles["en"], "Easy / Normal"; got != want {
		t.Fatalf("English title = %q, want %q", got, want)
	}
	if got, want := titles["ja"], "初級 / 中級"; got != want {
		t.Fatalf("Japanese title = %q, want %q", got, want)
	}
	if _, exists := titles["ko"]; exists {
		t.Fatal("unexpected empty Korean title")
	}
}
