### Airplay clone
##### Golang + OBS + ffmpeg + gRPC + http Portfolio project

- Clone the repository
- Run ``` go mod tidy```
- Open two terminals
- Run ``` go run main.go```
- Run OBS studio and start streaming to custom server rtmp://localhost:1935
- Run ``` go run player.go```

##### In the browser check url localhost:8080, you will get the video player. 

#### Make sure to run the project in above order. 

* Adaptive birrate needs high resource usage, so keep an eye on CPU and RAM and feel normal about your PC getting heated. 
