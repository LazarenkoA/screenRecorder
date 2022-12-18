package main

import (
	"context"
	"fmt"
	"image"
	"os"
	"time"

	"screenRecorder/screenshot"
	"screenRecorder/x264-go"
)

func main() {
	fps := 90
	screens := screen.InitDisplays(fps)

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	result := make([]chan image.Image, len(screens))
	for i, s := range screens {
		result[i] = s.StartRecord(ctx)
	}

	file, err := os.Create("example.264")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()

	opts := &x264.Options{
		Width:     screens[0].Bounds.Dx(),
		Height:    screens[0].Bounds.Dy(),
		FrameRate: 10,
		Tune:      "film",
		Preset:    "ultrafast",
		Profile:   "baseline",
		LogLevel:  x264.LogError,
	}
	enc, err := x264.NewEncoder(file, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	defer enc.Close()

	for img := range result[0] {
		enc.Encode(img)
	}
	enc.Flush()
}

func getImageFromFilePath(filePath string) image.Image {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	return image
}
