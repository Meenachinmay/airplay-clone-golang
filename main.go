package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/nareix/joy4/av/pubsub"
	"github.com/nareix/joy4/format/rtmp"
	"log"
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

func main() {
	a := app.New()
	w := a.NewWindow("RTMP Server")

	statusLabel := widget.NewLabel("Server is not running")

	startStopButton := widget.NewButton("Start Server", nil)

	var serverRunning bool

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
			serverRunning = false
			statusLabel.SetText("Server is not running")
			startStopButton.SetText("Start Server")
		}
	}

	content := container.NewVBox(
		statusLabel,
		startStopButton,
	)

	w.SetContent(content)

	w.Resize(fyne.NewSize(300, 100))
	w.ShowAndRun()
}
