.PHONY: setup run dry test build

setup:
	go mod tidy
	go run . setup

run:
	go run .

dry:
	go run . -dry

test:
	go test ./internal/...

# Single-file programs for Windows and Mac, no install needed on the target computer
build:
	go mod tidy
	mkdir -p dist
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o dist/anti-ozge-bot.exe .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "-s -w" -o dist/anti-ozge-bot-mac .
	@echo "Done: dist/anti-ozge-bot.exe (Windows) and dist/anti-ozge-bot-mac (Apple Silicon)"
