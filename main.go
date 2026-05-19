package main

import (
	"embed"
	"net"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func checkAndNotifyRunningInstance(app *App) (bool, error) {
	addr := "127.0.0.1:12115"
	
	// Try to listen on the single instance lock port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		// Port already bound -> Another instance is already running!
		if len(os.Args) > 1 {
			// Connect to the running instance and send the startup file path
			conn, errDial := net.DialTimeout("tcp", addr, 2*time.Second)
			if errDial == nil {
				defer conn.Close()
				_, _ = conn.Write([]byte(os.Args[1]))
			}
		}
		return true, nil
	}
	
	// We are the first/master instance. Start background TCP lock listener.
	go func() {
		defer listener.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 2048)
				n, err := c.Read(buf)
				if err == nil && n > 0 {
					filePath := string(buf[:n])
					app.HandleStartupFilePath(filePath)
				}
			}(conn)
		}
	}()
	
	return false, nil
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Check if a command line argument (file/folder path) was passed via context menu
	if len(os.Args) > 1 {
		app.startupFilePath = os.Args[1]
	}

	// Single instance guard
	hasInstance, err := checkAndNotifyRunningInstance(app)
	if err == nil && hasInstance {
		// Existing instance was notified, exit cleanly now
		os.Exit(0)
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:  "QBFile - Secure Direct Transfer",
		Width:  1024,
		Height: 768,
		MinWidth: 800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 20, B: 26, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
