# anti-ozge-bot

A WhatsApp bot that deflects Özge with smart reasoning, so you never have to.

## What it does

When Özge messages you, the bot steps in. It reads the message, works out the most reasonable way to close the conversation, and replies with a calm, logical answer that leaves nothing to follow up on.

| She sends | The bot |
|---|---|
| `?`, `??`, "online mısın", "orada mısın" | One calm reply, then calm silence during the cooldown |
| "baktın mı", "ne oldu", "any update?" | A reassuring status reply |
| "acil", "asap" | A calm "I'm prioritising it" |
| A new lead or assignment | A polite acknowledgement, and the lead is saved to `leads.jsonl` |
| Three more nudges during the cooldown | One final "I've seen everything, it's all in the queue" |

Also: it replies in her language (Turkish or English), never sends the same line twice in a row, waits a random 25–90 seconds and shows "typing…" before replying, and steps aside for 30 minutes whenever you reply to her yourself.

No AI, no paid APIs. Everything is scripted and runs on your machine.

## Proven track record

Özge has never messaged back after a deflection. Not once.

## Windows (no install needed)

1. Download `anti-ozge-bot.exe` from this repo's **Releases** page.
2. Put it in its own folder and double-click it.
3. Enter Özge's number and language, then scan the QR code in WhatsApp → Settings → Linked devices.

That's it. The program is a single file with everything built in; it stores `config.json`, `session.db` and `leads.jsonl` in the same folder. Close the window to stop it.

The first time, Windows may show "Windows protected your PC" because the file isn't code-signed. Click **More info → Run anyway**.

## Building it yourself

Everything below is only for building or developing; people who just use the `.exe` can skip it.

## Quick start (Mac / developers)

Requires Go 1.24+ (`brew install go`).

```bash
make setup   # downloads dependencies, asks for Özge's number and language
make dry     # test run: shows what it would reply, sends nothing
make run     # the real thing
```

On first run, scan the QR code: WhatsApp → Settings → Linked devices → Link a device. The session is saved in `session.db`, so you only scan once.

`make test` checks the message classifier offline. `make build` creates `dist/anti-ozge-bot.exe` for Windows and `dist/anti-ozge-bot-mac` for Apple Silicon Macs. Pushing a tag like `v1.0.0` builds both automatically on GitHub and attaches them to a Release. Stop the bot with `Ctrl+C` to see a session summary.

## Settings (`config.json`)

| Setting | Default | What it does |
|---|---|---|
| `ozge_number` | — | Her number with country code, digits only |
| `language` | `tr` | Reply language when hers can't be detected |
| `cooldown_minutes` | `20` | Calm silence to repeat nudges after a reply |
| `takeover_minutes` | `30` | How long the bot stays out after you reply yourself |
| `min_delay_seconds` / `max_delay_seconds` | `25` / `90` | Human-like reply delay |

Replies live in `internal/brain/brain.go`; edit them to sound more like you.

## Privacy

`config.json`, `session.db` and `leads.jsonl` stay on your machine and are excluded from git. Lead messages may contain patient details, so keep this repo private.

Built on [whatsmeow](https://github.com/tulir/whatsmeow), an unofficial WhatsApp client. Use it on your own account, at your own risk.
