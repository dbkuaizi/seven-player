package main

import (
	"embed"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := newLogger()

	app, err := NewApp(logger)
	if err != nil {
		logger.Error("failed to create app", "error", err)
		os.Exit(1)
	}

	err = wails.Run(&options.App{
		Title:            "PanPlayer 115",
		Width:            820,
		Height:           660,
		MinWidth:         800,
		MinHeight:        660,
		DisableResize:    false,
		Frameless:        false,
		BackgroundColour: &options.RGBA{R: 250, G: 250, B: 250, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: true,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		logger.Error("wails run failed", "error", err)
		os.Exit(1)
	}
}

func newLogger() *slog.Logger {
	writer := io.Writer(os.Stdout)

	configDir, err := os.UserConfigDir()
	if err == nil {
		logDir := filepath.Join(configDir, "panplayer")
		if mkErr := os.MkdirAll(logDir, 0o755); mkErr == nil {
			logFilePath := filepath.Join(logDir, "panplayer.log")
			logFile, openErr := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if openErr == nil {
				writer = io.MultiWriter(os.Stdout, logFile)
			}
		}
	}

	return slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
