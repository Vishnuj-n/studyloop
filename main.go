package main

import (
	"embed"
	"net/http"
	"strings"

	"ai-tutor/internal/app"
	"ai-tutor/internal/utils"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	a := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "Studyloop",
		Width:            1024,
		Height:           768,
		WindowStartState: options.Maximised,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "8f4e2a1b-9c3d-4e5f-b6a7-1c2d3e4f5a6b",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				// ponytail: focus existing window if user opens executable again
				if ctx := a.GetCtx(); ctx != nil {
					wailsruntime.WindowUnminimise(ctx)
					wailsruntime.Show(ctx)
				}
			},
		},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: notebookHandler(a),
		},
		BackgroundColour: &options.RGBA{R: 249, G: 249, B: 251, A: 255},
		OnStartup:        a.Startup,
		OnShutdown:       a.Shutdown,
		Bind: []interface{}{
			a,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func notebookHandler(a *app.App) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if a == nil {
			utils.Warnf("[notebookHandler] Service unavailable: app is nil")
			http.Error(rw, "notebook directory unavailable", http.StatusServiceUnavailable)
			return
		}
		uploadDir := a.GetNotebookUploadDir()
		if uploadDir == "" {
			utils.Warnf("[notebookHandler] Service unavailable: upload dir empty")
			http.Error(rw, "notebook directory unavailable", http.StatusServiceUnavailable)
			return
		}

		// Only handle requests under /notebooks/
		if !strings.HasPrefix(req.URL.Path, "/notebooks/") {
			return
		}

		// Serve only GET requests.
		if req.Method != http.MethodGet {
			utils.Warnf("[notebookHandler] Rejected method: %s", req.Method)
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		http.StripPrefix("/notebooks/", http.FileServer(http.Dir(uploadDir))).ServeHTTP(rw, req)
	})
}

