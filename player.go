package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const (
	rtmpURL  = "rtmp://localhost:1935"
	hlsPath  = "./hls"
	httpAddr = ":8080"
)

var qualities = []struct {
	name       string
	resolution string
	bitrate    string
}{
	{"480p", "854x480", "1M"},
	{"720p", "1280x720", "3M"},
	{"1080p", "1920x1080", "5M"},
}

func main() {
	if err := os.MkdirAll(hlsPath, os.ModePerm); err != nil {
		log.Fatal("Failed to create HLS directory:", err)
	}

	var wg sync.WaitGroup
	for _, q := range qualities {
		wg.Add(1)
		go func(quality struct {
			name       string
			resolution string
			bitrate    string
		}) {
			defer wg.Done()
			for {
				err := transcodeToHLS(quality)
				if err != nil {
					log.Printf("Transcoding failed for %s: %v\n", quality.name, err)
				}
			}
		}(q)
	}

	go createMasterPlaylist()

	http.HandleFunc("/hls/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request for: %s\n", r.URL.Path)
		http.StripPrefix("/hls/", http.FileServer(http.Dir(hlsPath))).ServeHTTP(w, r)
	})

	http.HandleFunc("/", servePlayer)

	fmt.Printf("Server listening on %s\n", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func createMasterPlaylist() {
	master := "#EXTM3U\n#EXT-X-VERSION:3\n"
	for _, q := range qualities {
		master += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%s,RESOLUTION=%s\n%s.m3u8\n",
			strings.TrimRight(q.bitrate, "M")+"000", q.resolution, q.name)
	}

	masterPath := filepath.Join(hlsPath, "master.m3u8")
	if err := os.WriteFile(masterPath, []byte(master), 0644); err != nil {
		log.Printf("Error writing master playlist: %v\n", err)
	}
}

func transcodeToHLS(q struct {
	name       string
	resolution string
	bitrate    string
}) error {
	outputPath := filepath.Join(hlsPath, q.name+".m3u8")
	log.Printf("Transcoding %s to %s\n", q.name, outputPath)

	return ffmpeg.Input(rtmpURL).
		Output(outputPath, ffmpeg.KwArgs{
			"c:v":           "libx264",
			"c:a":           "aac",
			"b:v":           q.bitrate,
			"s":             q.resolution,
			"hls_time":      "4",
			"hls_list_size": "5",
			"hls_flags":     "delete_segments+independent_segments",
			"format":        "hls",
		}).
		OverWriteOutput().
		ErrorToStdOut().
		Run()
}

func servePlayer(w http.ResponseWriter, r *http.Request) {
	playerHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>Adaptive Bitrate HLS Player</title>
    <script src="https://cdn.jsdelivr.net/npm/hls.js@latest"></script>
    <style>
        #quality-controls {
            margin-top: 10px;
        }
        button {
            margin-right: 10px;
        }
    </style>
</head>
<body>
    <video id="video" controls style="width: 640px; height: 360px;"></video>
    <div id="quality-controls">
        <button onclick="setQuality(2)">1080p</button>
        <button onclick="setQuality(1)">720p</button>
        <button onclick="setQuality(0)">480p</button>
        <button onclick="setQuality(-1)">Auto</button>
    </div>
    <script>
        var video = document.getElementById('video');
        var hls = new Hls();
        hls.loadSource('/hls/master.m3u8');
        hls.attachMedia(video);
        hls.on(Hls.Events.MANIFEST_PARSED, function() {
            video.play();
        });

        function setQuality(quality) {
            hls.currentLevel = quality;
        }
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, playerHTML)
}
