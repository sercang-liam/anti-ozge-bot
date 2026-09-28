package brain

import "testing"

func TestClassify(t *testing.T) {
	cases := map[string]Kind{
		"?":                                  Nudge,
		"???":                                Nudge,
		"online mısın":                       Nudge,
		"orada mısın?":                       Nudge,
		"are you there?":                     Nudge,
		"baktın mı":                          Status,
		"ne oldu?":                           Status,
		"any update?":                        Status,
		"acil bakar mısın":                   Urgent,
		"Yeni lead: Anna K. +44 7700 900123": Lead,
		"Sana bir hasta atadım":              Lead,
		"Günaydın":                           Greeting,
		"Toplantı saat 3te":                  Other,
	}
	for text, want := range cases {
		if got := Classify(text); got != want {
			t.Errorf("Classify(%q) = %s, want %s", text, got, want)
		}
	}
}

func TestDetectLang(t *testing.T) {
	if DetectLang("orada mısın?", "en") != "tr" {
		t.Error("expected tr")
	}
	if DetectLang("are you online?", "tr") != "en" {
		t.Error("expected en")
	}
	if DetectLang("???", "tr") != "tr" {
		t.Error("expected fallback tr")
	}
}

func TestPickerDoesNotRepeat(t *testing.T) {
	p := NewPicker()
	prev := ""
	for i := 0; i < 50; i++ {
		r := p.Reply(Nudge, "tr")
		if r == prev {
			t.Fatalf("same reply twice in a row: %q", r)
		}
		prev = r
	}
}
