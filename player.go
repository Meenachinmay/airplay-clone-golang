package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const (
	rtmpURL  = "rtmp://localhost:1935"
	hlsPath  = "./hls"
	httpAddr = ":8080"
)

func main() {
	if err := os.MkdirAll(hlsPath, os.ModePerm); err != nil {
		log.Fatal("Failed to create HLS directory:", err)
	}

	go func() {
		for {
			log.Println("Starting transcoding process...")
			err := transcodeToHLS()
			if err != nil {
				log.Println("Transcoding failed:", err)
				time.Sleep(5 * time.Second) // Wait before retrying
			}
		}
	}()

	http.HandleFunc("/hls/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request for: %s\n", r.URL.Path)
		http.StripPrefix("/hls/", http.FileServer(http.Dir(hlsPath))).ServeHTTP(w, r)
	})

	http.HandleFunc("/", servePlayer)

	fmt.Printf("Server listening on %s\n", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func transcodeToHLS() error {
	output := filepath.Join(hlsPath, "stream.m3u8")
	log.Printf("Transcoding RTMP stream from %s to HLS at %s\n", rtmpURL, output)

	err := ffmpeg.Input(rtmpURL).
		Output(output, ffmpeg.KwArgs{
			"c:v":           "libx264",
			"c:a":           "aac",
			"b:v":           "1M",
			"b:a":           "128k",
			"hls_time":      "10",
			"hls_list_size": "6",
			"hls_flags":     "delete_segments",
			"format":        "hls",
		}).
		OverWriteOutput().
		ErrorToStdOut().
		Run()

	if err != nil {
		return fmt.Errorf("ffmpeg transcoding failed: %w", err)
	}

	log.Println("Transcoding completed successfully")
	return nil
}

func servePlayer(w http.ResponseWriter, r *http.Request) {
	playerHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>HLS Player</title>
    <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
</head>
<body>
    <video id="video" controls style="width: 640px; height: 360px;"></video>
    <script>
        var video = document.getElementById('video');
        var videoSrc = '/hls/stream.m3u8';
        if (Hls.isSupported()) {
            var hls = new Hls();
            hls.loadSource(videoSrc);
            hls.attachMedia(video);
            hls.on(Hls.Events.MANIFEST_PARSED, function() {
                video.play();
            });
        }
        else if (video.canPlayType('application/vnd.apple.mpegurl')) {
            video.src = videoSrc;
            video.addEventListener('loadedmetadata', function() {
                video.play();
            });
        }
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, playerHTML)
}
