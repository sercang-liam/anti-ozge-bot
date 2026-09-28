<p align="center">
  <img src="icon.png" width="128" alt="anti-ozge-bot icon">
</p>

<h1 align="center">anti-ozge-bot</h1>

<p align="center">A WhatsApp bot that deflects Özge with smart reasoning, so you never have to.</p>

<p align="center">
  <a href="https://github.com/sercang-liam/anti-ozge-bot/releases/latest/download/anti-ozge-bot.exe"><img src="assets/download-windows.svg" alt="Download for Windows" height="56"></a>
  &nbsp;
  <a href="https://github.com/sercang-liam/anti-ozge-bot/releases/latest/download/anti-ozge-bot-mac"><img src="assets/download-mac.svg" alt="Download for Mac" height="56"></a>
</p>

<p align="center"><sub>Both buttons always download the newest version.</sub></p>

## What it does

When Özge messages you, the bot steps in. It reads the message, works out the most reasonable way to close the conversation, and replies with a calm, logical answer that leaves nothing to follow up on.

| She sends | The bot |
|---|---|
| `?`, `??`, "online mısın", "orada mısın" | A calm "I'm here, on it" |
| "baktın mı", "ne oldu", "any update?" | A reassuring status reply |
| "acil", "asap" | A calm "I'm prioritising it" |
| A new lead or assignment | A polite acknowledgement, and the lead is saved to `leads.jsonl` |
| "günaydın", "merhaba" | A friendly greeting back |
| Anything else, including photos and voice notes | A short "noted, thanks" |

Also: it replies in her language (Turkish or English), never sends the same line twice in a row, answers every message within 2–5 seconds (showing "typing…" first; several messages in a row get one reply), and keeps answering even when you write to her yourself.

No AI, no paid APIs. Everything is scripted and runs on your machine.

## Proven track record

Özge has never messaged back after a deflection. Not once.

## Windows (no install needed)

1. Click **Download for Windows** at the top of this page.
2. Put it in its own folder and double-click it.
3. Enter Özge's number and language, then scan the QR code in WhatsApp → Settings → Linked devices.

That's it. The program is a single file with everything built in; it stores `config.json`, `session.db` and `leads.jsonl` in the same folder. Close the window to stop it.

The first time, Windows may show "Windows protected your PC" because the file isn't code-signed. Click **More info → Run anyway**.

## Mac

Click **Download for Mac**, then in Terminal:

```bash
cd ~/Downloads
chmod +x anti-ozge-bot-mac
./anti-ozge-bot-mac
```

If macOS says it can't verify the developer, open **System Settings → Privacy & Security**, scroll down and click **Open Anyway**.

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

Replies live in `internal/brain/brain.go`; edit them to sound more like you.

## Privacy

`config.json`, `session.db` and `leads.jsonl` stay on your machine and are excluded from git. Lead messages may contain patient details, so keep this repo private.

Built on [whatsmeow](https://github.com/tulir/whatsmeow), an unofficial WhatsApp client. Use it on your own account, at your own risk.
