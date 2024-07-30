package main

import (
	"log"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
)

var (
	server *rtmp.Server
	queue  *pubsub.Queue
)

func startRTMPServer() error {
	server = &rtmp.Server{}
	queue = pubsub.NewQueue()

	server.HandlePublish = func(conn *rtmp.Conn) {
		log.Println("New stream:", conn.URL.Path)
		streams, _ := conn.Streams()

		if err := queue.WriteHeader(streams); err != nil {
			log.Println("Error writing header:", err)
			return
		}

		for {
			packet, err := conn.ReadPacket()
			if err != nil {
				break
			}
			if err := queue.WritePacket(packet); err != nil {
				log.Println("Error writing packet:", err)
				break
			}
		}
	}

	server.HandlePlay = func(conn *rtmp.Conn) {
		log.Println("New viewer:", conn.URL.Path)
		cursor := queue.Latest()

		streams, err := cursor.Streams()
		if err != nil {
			log.Println("Error getting streams:", err)
			return
		}

		conn.WriteHeader(streams)

		for {
			packet, err := cursor.ReadPacket()
			if err != nil {
				break
			}
			conn.WritePacket(packet)
		}
	}

	return server.ListenAndServe()
}

func startHTTPServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	log.Println("Starting HTTP server on :8000")
	go func() {
		if err := http.ListenAndServe(":8000", nil); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()
}

func main() {
	// Start the HTTP server
	startHTTPServer()

	// Create a new Fyne application
	a := app.New()
	w := a.NewWindow("RTMP Server")

	// Create a label to show server status
	statusLabel := widget.NewLabel("Server is not running")

	// Declare the button
	startStopButton := widget.NewButton("Start Server", nil)

	// Create a variable to track server state
	var serverRunning bool

	// Set the OnTapped function for the button
	startStopButton.OnTapped = func() {
		if !serverRunning {
			go func() {
				if err := startRTMPServer(); err != nil {
					log.Printf("Error starting server: %v", err)
					statusLabel.SetText("Error starting server")
				}
			}()
			serverRunning = true
			statusLabel.SetText("Server is running")
			startStopButton.SetText("Stop Server")
		} else {
			// Note: This is a simplified stop. In a real application,
			// you'd want to properly shut down the RTMP server.
			serverRunning = false
			statusLabel.SetText("Server is not running")
			startStopButton.SetText("Start Server")
		}
	}

	// Create a container with the label and button
	content := container.NewVBox(
		statusLabel,
		startStopButton,
	)

	// Set the content of the window
	w.SetContent(content)

	// Show and run the application
	w.Resize(fyne.NewSize(300, 100))
	w.ShowAndRun()
}
