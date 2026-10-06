package main

import (
	"embed"
	"net/http"
	"os"
	"strings"

	"ai-tutor/internal/app"
	"ai-tutor/internal/utils"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	_ = godotenv.Load()

	// Create an instance of the app structure
	a := app.NewApp()

	lockID := "8f4e2a1b-9c3d-4e5f-b6a7-1c2d3e4f5a6b"
	if os.Getenv("APP_ENV") == "dev" {
		lockID = "studyloop-dev-instance-lock"
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:            "Studyloop",
		Width:            1024,
		Height:           768,
		WindowStartState: options.Maximised,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: lockID,
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
			Handler: customAssetHandler(a),
		},
		BackgroundColour: &options.RGBA{R: 249, G: 249, B: 251, A: 255},
		OnStartup:        a.Startup,
		OnShutdown:       a.Shutdown,
		Debug: options.Debug{
			OpenInspectorOnStartup: false,
		},
		Bind: []interface{}{
			a,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func customAssetHandler(a *app.App) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/notebooks/") {
			if a == nil {
				utils.Warnf("[customAssetHandler] Service unavailable: app is nil")
				http.Error(rw, "notebook directory unavailable", http.StatusServiceUnavailable)
				return
			}
			uploadDir := a.GetNotebookUploadDir()
			if uploadDir == "" {
				utils.Warnf("[customAssetHandler] Service unavailable: upload dir empty")
				http.Error(rw, "notebook directory unavailable", http.StatusServiceUnavailable)
				return
			}

			if req.Method != http.MethodGet {
				utils.Warnf("[customAssetHandler] Rejected method: %s", req.Method)
				rw.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			fs := noDirListingFS{fs: http.Dir(uploadDir)}
			http.StripPrefix("/notebooks/", http.FileServer(fs)).ServeHTTP(rw, req)
			return
		}

		if strings.HasPrefix(req.URL.Path, "/note-assets/") || strings.HasPrefix(req.URL.Path, "/notes/") {
			if a == nil {
				utils.Warnf("[customAssetHandler] Service unavailable: app is nil")
				http.Error(rw, "notes directory unavailable", http.StatusServiceUnavailable)
				return
			}
			notesDir := a.GetNotesDir()
			if notesDir == "" {
				utils.Warnf("[customAssetHandler] Service unavailable: notes dir empty")
				http.Error(rw, "notes directory unavailable", http.StatusServiceUnavailable)
				return
			}

			if req.Method != http.MethodGet {
				utils.Warnf("[customAssetHandler] Rejected method: %s", req.Method)
				rw.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			prefix := "/notes/"
			if strings.HasPrefix(req.URL.Path, "/note-assets/") {
				prefix = "/note-assets/"
			}

			fs := noDirListingFS{fs: http.Dir(notesDir)}
			http.StripPrefix(prefix, http.FileServer(fs)).ServeHTTP(rw, req)
			return
		}
	})
}

type noDirListingFS struct {
	fs http.FileSystem
}

func (nfs noDirListingFS) Open(name string) (http.File, error) {
	file, err := nfs.fs.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := file.Stat()
	if err != nil {
		_ = file.Close() // read-only file handle being aborted
		return nil, err
	}
	if stat.IsDir() {
		_ = file.Close() // prevent directory listing
		return nil, os.ErrPermission
	}
	return file, nil
}

