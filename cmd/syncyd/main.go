package main

import (
	"log/slog"
	"os"
)

func main() {
	slogOpts := &slog.HandlerOptions{}
	ll := slog.NewJSONHandler(os.Stderr, slogOpts)
}
