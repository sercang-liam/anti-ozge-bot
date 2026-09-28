// Package brain decides what kind of message Özge sent and picks a calm reply.
// It has no external dependencies, so it can be tested on its own.
package brain

import (
	"math/rand"
	"regexp"
	"strings"
)

// Kind is the category of an incoming message.
type Kind string

const (
	Nudge    Kind = "nudge"    // "?", "online mısın", "are you there"
	Status   Kind = "status"   // "baktın mı", "ne oldu", "any update"
	Urgent   Kind = "urgent"   // "acil", "asap"
	Lead     Kind = "lead"     // a new lead or assignment
	Greeting Kind = "greeting" // "günaydın", "merhaba"
	Other    Kind = "other"    // anything else
	Closer   Kind = "closer"   // used when she keeps nudging during the cooldown
)

var (
	onlyPunct = regexp.MustCompile(`^[\s?？!.¿…]+$`)
	phone     = regexp.MustCompile(`\+?\d[\d\s().-]{7,}\d`)

	leadWords = []string{
		"lead", "patient", "assigned", "hasta", "atadım", "atadim", "atandı", "atandi",
		"numara", "number:", "name:", "isim:", "yeni kayıt", "yeni kayit",
	}
	urgentWords = []string{"acil", "urgent", "asap", "hemen", "çabuk", "cabuk", "immediately"}
	statusWords = []string{
		"baktın mı", "baktin mi", "bakabildin", "ne oldu", "ne durumda", "haber var",
		"dönüş", "donus", "gelişme", "gelisme", "any update", "update?", "any news", "status",
	}
	nudgeWords = []string{
		"online", "orada mısın", "orada misin", "orda mısın", "orda misin", "burada mısın",
		"burada misin", "burda mısın", "burda misin", "müsait", "musait", "are you there",
		"you there", "hello?", "hey?", "alo",
	}
	greetingWords = []string{"günaydın", "gunaydin", "merhaba", "selam", "good morning", "hello", "hi ", "hey"}

	trHints = []string{"ç", "ğ", "ı", "ö", "ş", "ü", " mi ", " mı ", " mi?", " mı?", "misin", "mısın", "merhaba", "selam", "tamam", "lütfen", "lutfen", "acil", "hasta", "baktin", "gunaydin", "orda", "orada"}
	enHints = []string{"are you", "please", "hello", "update", "patient", "urgent", "morning", "there", "you"}
)

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// Classify returns the kind of message.
func Classify(text string) Kind {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" || onlyPunct.MatchString(t) {
		return Nudge
	}
	if phone.MatchString(t) || containsAny(t, leadWords) {
		return Lead
	}
	if containsAny(t, urgentWords) {
		return Urgent
	}
	short := len([]rune(t)) <= 60
	switch {
	case short && containsAny(t, statusWords):
		return Status
	case short && containsAny(t, nudgeWords):
		return Nudge
	case short && containsAny(t+" ", greetingWords):
		return Greeting
	}
	return Other
}

// DetectLang guesses "tr" or "en" from the text, falling back to the given default.
func DetectLang(text, fallback string) string {
	t := " " + strings.ToLower(text) + " "
	if containsAny(t, trHints) {
		return "tr"
	}
	if containsAny(t, enHints) {
		return "en"
	}
	if fallback == "en" {
		return "en"
	}
	return "tr"
}

var replies = map[string]map[Kind][]string{
	"tr": {
		Lead: {
			"Aldım, ilgileniyorum. Gelişme olunca buradan yazarım.",
			"Tamamdır, lead bende. Sonuç olunca haber veririm.",
			"Gördüm, üzerindeyim. Netleşen bir şey olunca dönüş yaparım.",
		},
		Nudge: {
			"Buradayım, elimdeki işlerle ilgileniyorum. Yeni bir gelişme olunca ilk sana yazarım.",
			"Mesajlarını gördüm. Hepsi üzerinde çalışıyorum, netleşince haber veririm.",
			"Buradayım, sırayla bakıyorum. Gelişme olunca yazarım.",
		},
		Status: {
			"Henüz netleşen bir şey yok; olduğunda ilk sana yazacağım.",
			"Takipteyim. Bir gelişme olunca buradan paylaşırım.",
			"Üzerinde çalışıyorum, sonuç çıkınca haber veririm.",
		},
		Urgent: {
			"Anladım, öne alıyorum. Netleşince hemen yazarım.",
			"Tamam, öncelik veriyorum. Gelişme olunca haber veririm.",
		},
		Greeting: {
			"Merhaba, kolay gelsin.",
			"Selam, kolay gelsin.",
		},
		Other: {
			"Tamamdır, not aldım.",
			"Anladım, teşekkürler.",
			"Tamam, not ettim. Gerekirse dönüş yaparım.",
		},
		Closer: {
			"Tüm mesajlarını gördüm, hepsi sırada. Bir gelişme olduğunda ilk sana yazacağım.",
		},
	},
	"en": {
		Lead: {
			"Got it, I'm on it. I'll update you here when there's news.",
			"Thanks, the lead is with me. I'll let you know once there's an outcome.",
		},
		Nudge: {
			"I'm here, working through everything. You'll be the first to know when something changes.",
			"Seen your messages and I'm on them. I'll update you as soon as there's news.",
		},
		Status: {
			"Nothing new yet; I'll message you as soon as there is.",
			"Still on it. I'll share an update here when there's progress.",
		},
		Urgent: {
			"Understood, I'm prioritising it. I'll update you as soon as it's clear.",
		},
		Greeting: {
			"Hi, hope your day's going well.",
		},
		Other: {
			"Noted, thanks.",
			"Got it, thanks.",
		},
		Closer: {
			"I've seen all your messages and they're all in the queue. I'll message you first as soon as there's news.",
		},
	},
}

// Picker chooses replies without repeating the previous one for the same kind.
type Picker struct {
	last map[string]int
}

// NewPicker returns a ready Picker.
func NewPicker() *Picker { return &Picker{last: map[string]int{}} }

// Reply returns a calm reply for the given kind and language.
func (p *Picker) Reply(kind Kind, lang string) string {
	set, ok := replies[lang]
	if !ok {
		set = replies["tr"]
	}
	options, ok := set[kind]
	if !ok || len(options) == 0 {
		options = set[Other]
	}
	key := lang + ":" + string(kind)
	i := rand.Intn(len(options))
	if prev, seen := p.last[key]; seen && len(options) > 1 && i == prev {
		i = (i + 1) % len(options)
	}
	p.last[key] = i
	return options[i]
}
