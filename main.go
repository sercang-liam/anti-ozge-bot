package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	_ "modernc.org/sqlite" // pure-Go SQLite: no C compiler, builds a single .exe

	"anti-ozge-bot/internal/brain"
)

const (
	leadsFile     = "leads.jsonl"
	maxMessageAge = 10 * time.Minute // ignore old messages delivered at startup
	leadCooldown  = 2 * time.Minute  // leads and urgent messages get answered more often
	closerAfter   = 3                // nudges during cooldown before one closing reply
)

type Bot struct {
	cfg    Config
	client *whatsmeow.Client
	picker   *brain.Picker
	dryRun   bool
	selfTest bool // treat your own "Message yourself" chat as Özge, for testing

	mu          sync.Mutex
	lastReply   time.Time
	lastManual  time.Time
	pending     bool
	nudgeCount  int
	closerSent  bool
	sentByBot   map[types.MessageID]bool
	received    int
	replied     int
	ignored     int
	leadsSaved  int
}

func logf(format string, a ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, a...))
}

// isOzge checks the given addresses against her number, resolving hidden (@lid) ids.
func (b *Bot) isOzge(ctx context.Context, jids ...types.JID) bool {
	for _, j := range jids {
		if j.IsEmpty() {
			continue
		}
		user := j.User
		if j.Server == types.HiddenUserServer {
			if pn, err := b.client.Store.LIDs.GetPNForLID(ctx, j); err == nil && !pn.IsEmpty() {
				user = pn.User
			}
		}
		if onlyDigits(user) == b.cfg.OzgeNumber {
			return true
		}
	}
	return false
}

// isSelfChat reports whether the chat is your own "Message yourself" chat.
func (b *Bot) isSelfChat(ctx context.Context, chat types.JID) bool {
	if b.client.Store.ID == nil {
		return false
	}
	own := b.client.Store.ID.User
	if chat.Server == types.HiddenUserServer {
		pn, err := b.client.Store.LIDs.GetPNForLID(ctx, chat)
		return err == nil && !pn.IsEmpty() && pn.User == own
	}
	return chat.User == own
}

func messageText(m *waE2E.Message) string {
	if m == nil {
		return ""
	}
	for _, t := range []string{
		m.GetConversation(),
		m.GetExtendedTextMessage().GetText(),
		m.GetImageMessage().GetCaption(),
		m.GetVideoMessage().GetCaption(),
		m.GetDocumentMessage().GetCaption(),
	} {
		if t != "" {
			return t
		}
	}
	return ""
}

func (b *Bot) saveLead(text string) {
	f, err := os.OpenFile(leadsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		logf("   could not save lead: %v", err)
		return
	}
	defer f.Close()
	line, _ := json.Marshal(map[string]string{"at": time.Now().Format(time.RFC3339), "text": text})
	f.Write(append(line, '\n'))
	logf("   📌 Lead saved to %s so nothing gets lost.", leadsFile)
}

func (b *Bot) handle(evt any) {
	switch v := evt.(type) {
	case *events.Message:
		go b.onMessage(v)
	case *events.Connected:
		mode := ""
		if b.dryRun {
			mode = " (DRY RUN: nothing will be sent)"
		}
		logf("✅ anti-ozge-bot is running%s. Ctrl+C to stop.", mode)
	case *events.LoggedOut:
		logf("⚠️  Logged out from WhatsApp. Delete session.db and run again to relink.")
	}
}

func (b *Bot) onMessage(v *events.Message) {
	ctx := context.Background()
	info := v.Info
	if info.IsGroup || info.Chat.Server == types.BroadcastServer {
		return
	}
	if v.Message.GetReactionMessage() != nil || v.Message.GetProtocolMessage() != nil {
		return
	}

	if info.IsFromMe {
		b.mu.Lock()
		own := b.sentByBot[info.ID]
		b.mu.Unlock()
		if own {
			return
		}
		if b.selfTest && b.isSelfChat(ctx, info.Chat) {
			// Self-test: a message you wrote to yourself is handled as if Özge sent it.
		} else {
			// You wrote to her yourself, so the bot steps aside for a while.
			if b.isOzge(ctx, info.Chat, info.RecipientAlt) {
				b.mu.Lock()
				b.lastManual = time.Now()
				b.mu.Unlock()
				logf("✋ You replied yourself, the bot stays out of this chat for now.")
			}
			return
		}
	} else if !b.isOzge(ctx, info.Sender, info.SenderAlt, info.Chat) {
		return
	}
	if time.Since(info.Timestamp) > maxMessageAge {
		return
	}

	text := messageText(v.Message)
	kind := brain.Other
	shown := "[media]"
	if text != "" {
		kind = brain.Classify(text)
		shown = text
	}
	lang := brain.DetectLang(text, b.cfg.Language)
	if len([]rune(shown)) > 120 {
		shown = string([]rune(shown)[:120]) + "…"
	}

	b.mu.Lock()
	b.received++
	b.mu.Unlock()
	logf("📩 Özge (%s): %s", kind, shown)
	if kind == brain.Lead {
		b.saveLead(text)
		b.mu.Lock()
		b.leadsSaved++
		b.mu.Unlock()
	}

	replyKind, reason := b.decide(kind)
	if replyKind == "" {
		logf("   ↳ %s", reason)
		return
	}
	b.reply(ctx, info.Chat, replyKind, lang)
}

// decide returns the kind of reply to send, or "" with a reason to stay quiet.
func (b *Bot) decide(kind brain.Kind) (brain.Kind, string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	cooldown := time.Duration(b.cfg.CooldownMinutes) * time.Minute
	takeover := time.Duration(b.cfg.TakeoverMinutes) * time.Minute

	skip := func(reason string) (brain.Kind, string) {
		b.ignored++
		return "", reason
	}
	switch {
	case now.Sub(b.lastManual) < takeover:
		return skip("You are handling this chat, no auto-reply.")
	case b.pending:
		return skip("A reply is already on its way.")
	case kind == brain.Lead || kind == brain.Urgent:
		if now.Sub(b.lastReply) < leadCooldown {
			return skip("Answered a moment ago, staying calm.")
		}
	case now.Sub(b.lastReply) < cooldown:
		b.nudgeCount++
		if b.nudgeCount >= closerAfter && !b.closerSent {
			b.closerSent = true
			b.pending = true
			return brain.Closer, ""
		}
		return skip("Already answered recently. Staying calm and quiet.")
	}
	b.nudgeCount = 0
	b.closerSent = false
	b.pending = true
	return kind, ""
}

func (b *Bot) reply(ctx context.Context, chat types.JID, kind brain.Kind, lang string) {
	defer func() {
		b.mu.Lock()
		b.pending = false
		b.mu.Unlock()
	}()

	text := b.picker.Reply(kind, lang)
	minD, maxD := b.cfg.MinDelaySeconds, b.cfg.MaxDelaySeconds
	time.Sleep(time.Duration(minD+rand.Intn(maxD-minD+1)) * time.Second)

	if b.dryRun {
		logf("   ↳ [dry run] would reply: %s", text)
	} else {
		_ = b.client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
		time.Sleep(time.Duration(2+rand.Intn(4)) * time.Second)
		_ = b.client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)
		resp, err := b.client.SendMessage(ctx, chat, &waE2E.Message{Conversation: proto.String(text)})
		if err != nil {
			logf("   ↳ ❌ Could not send: %v", err)
			return
		}
		b.mu.Lock()
		b.sentByBot[resp.ID] = true
		b.mu.Unlock()
		logf("   ↳ 🤖 Replied: %s", text)
	}

	b.mu.Lock()
	b.lastReply = time.Now()
	b.replied++
	b.mu.Unlock()
}

// fatal prints an error and keeps the window open, so a double-clicked .exe doesn't just vanish.
func fatal(format string, a ...any) {
	fmt.Printf("\n❌ "+format+"\n", a...)
	fmt.Print("\nPress Enter to close.")
	bufio.NewReader(os.Stdin).ReadString('\n')
	os.Exit(1)
}

// useExeFolder keeps config.json, session.db and leads.jsonl next to the program,
// no matter where it was launched from (skipped for "go run", which builds in a temp folder).
func useExeFolder() {
	exe, err := os.Executable()
	if err != nil || strings.Contains(exe, "go-build") {
		return
	}
	_ = os.Chdir(filepath.Dir(exe))
}

func main() {
	dry := flag.Bool("dry", false, "print replies instead of sending them")
	selfTest := flag.Bool("selftest", false, "treat your own \"Message yourself\" chat as Özge")
	flag.Parse()
	prepareConsole()
	useExeFolder()

	if flag.Arg(0) == "setup" {
		runSetup()
		return
	}
	cfg, err := loadConfig()
	if errors.Is(err, fs.ErrNotExist) {
		runSetup() // first launch: ask the two questions, then start
		cfg, err = loadConfig()
	}
	if err != nil {
		fatal("config.json is not valid (%v). Delete it and start again.", err)
	}

	color := runtime.GOOS != "windows"
	ctx := context.Background()
	db, err := sql.Open("sqlite", "file:session.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		fatal("Could not open session.db: %v", err)
	}
	db.SetMaxOpenConns(1)
	container := sqlstore.NewWithDB(db, "sqlite3", waLog.Stdout("Database", "WARN", color))
	if err := container.Upgrade(ctx); err != nil {
		fatal("Could not prepare session.db: %v", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		fatal("Could not load the WhatsApp session: %v", err)
	}
	client := whatsmeow.NewClient(device, waLog.Stdout("Client", "WARN", color))

	bot := &Bot{cfg: cfg, client: client, picker: brain.NewPicker(), dryRun: *dry, selfTest: *selfTest, sentByBot: map[types.MessageID]bool{}}
	if *selfTest {
		logf("🧪 Self-test mode: messages in your own \"Message yourself\" chat count as Özge.")
	}
	client.AddEventHandler(bot.handle)

	logf("Starting WhatsApp...")
	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(ctx)
		if err := client.Connect(); err != nil {
			fatal("Could not connect to WhatsApp: %v", err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("\nScan in WhatsApp → Settings → Linked devices → Link a device:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				logf("Login: %s", evt.Event)
			}
		}
	} else if err := client.Connect(); err != nil {
		fatal("Could not connect to WhatsApp: %v", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	bot.mu.Lock()
	fmt.Printf("\nSession summary: %d messages from Özge, %d replies, %d calmly ignored, %d leads saved.\n",
		bot.received, bot.replied, bot.ignored, bot.leadsSaved)
	bot.mu.Unlock()
	client.Disconnect()
}
