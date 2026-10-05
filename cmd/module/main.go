package main

import (
	"log/slog"
	"os"

	"github.com/Muxcore-Media/ai-runtime/internal"
	modulesdk "github.com/Muxcore-Media/core/sdk/go/module"
)

func main() {
	mod := internal.NewModule()
	insecure := os.Getenv("MUXCORE_INSECURE_DISABLE_TLS") == "true" || os.Getenv("MUXCORE_GRPC_INSECURE") == "true"
	if err := modulesdk.Run(modulesdk.Config{
		Module:   mod,
		Insecure: insecure,
	}); err != nil {
		slog.Error("module exited", "error", err)
		os.Exit(1)
	}
}
