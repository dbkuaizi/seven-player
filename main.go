package main

import (
	"embed"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := newLogger()

	service, err := NewApp(logger)
	if err != nil {
		logger.Error("failed to create app", "error", err)
		os.Exit(1)
	}

	app := application.New(application.Options{
		Name:        "PanPlayer 115",
		Description: "PanPlayer 115",
		Logger:      logger,
		Services: []application.Service{
			application.NewService(service),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	service.window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "PanPlayer 115",
		Width:            820,
		Height:           660,
		MinWidth:         800,
		MinHeight:        660,
		DisableResize:    false,
		Frameless:        false,
		BackgroundColour: application.NewRGBA(250, 250, 250, 255),
		EnableFileDrop:   true,
		URL:              "/",
	})

	if err := app.Run(); err != nil {
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
