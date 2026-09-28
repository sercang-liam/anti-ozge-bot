package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const configFile = "config.json"

// Config holds the bot settings, stored in config.json.
type Config struct {
	OzgeNumber      string `json:"ozge_number"`       // digits only, with country code
	Language        string `json:"language"`          // fallback reply language: tr or en
	CooldownMinutes int    `json:"cooldown_minutes"`  // silence to repeat nudges after a reply
	TakeoverMinutes int    `json:"takeover_minutes"`  // bot stays out after you reply yourself
	MinDelaySeconds int    `json:"min_delay_seconds"` // human-like delay before replying
	MaxDelaySeconds int    `json:"max_delay_seconds"`
}

func defaultConfig() Config {
	return Config{Language: "tr", CooldownMinutes: 20, TakeoverMinutes: 30, MinDelaySeconds: 25, MaxDelaySeconds: 90}
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func loadConfig() (Config, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	cfg.OzgeNumber = onlyDigits(cfg.OzgeNumber)
	if cfg.OzgeNumber == "" {
		return cfg, fmt.Errorf("ozge_number is empty")
	}
	if cfg.MaxDelaySeconds < cfg.MinDelaySeconds {
		cfg.MaxDelaySeconds = cfg.MinDelaySeconds
	}
	return cfg, nil
}

func runSetup() {
	in := bufio.NewReader(os.Stdin)
	ask := func(q string) string {
		fmt.Print(q)
		line, _ := in.ReadString('\n')
		return strings.TrimSpace(line)
	}

	fmt.Println("\nanti-ozge-bot setup")
	cfg := defaultConfig()
	cfg.OzgeNumber = onlyDigits(ask("Özge's WhatsApp number with country code (e.g. 905551112233): "))
	if cfg.OzgeNumber == "" {
		fatal("A number is required. Start the program again to retry.")
	}
	if strings.ToLower(ask("Default reply language, tr or en [tr]: ")) == "en" {
		cfg.Language = "en"
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configFile, data, 0o600); err != nil {
		fatal("Could not save config.json: %v", err)
	}
	fmt.Println("\nSaved to config.json.")
}
