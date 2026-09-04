package logging

import (
	"io"
	"log/slog"
	"os"
)

func New(level slog.Level, out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	options := &slog.HandlerOptions{Level: level, AddSource: true}
	return slog.New(slog.NewJSONHandler(out, options))
}
