package screen

import (
	"bufio"
	"bytes"
	"context"
	"image"
	"image/color"
	"time"

	"github.com/kbinani/screenshot"
	"screenRecorder/x264-go"
)

type Screen struct {
	id     int
	fps    int
	Bounds image.Rectangle
	codec  ICodec
	buff   bytes.Buffer
}

type ICodec interface {
	Encode(im image.Image) (err error)
	Flush() (err error)
}

func InitDisplays(fps int) (result []*Screen) {
	if fps > 100 {
		fps = 100
	}

	n := screenshot.NumActiveDisplays()
	for i := 0; i < n; i++ {
		result = append(result, &Screen{id: i, fps: fps, Bounds: screenshot.GetDisplayBounds(i)})
		result[len(result)-1].initCodec()
	}

	return result
}

func (s *Screen) StartRecord(ctx context.Context) chan []byte {
	result := make(chan []byte)
	go s.start(ctx, result)

	return result

	// fileName := fmt.Sprintf("%d_%dx%d.jpg", i, bounds.Dx(), bounds.Dy())
	// file, _ := os.Create(fileName)
	// defer file.Close()
	// png.Encode(file, img)
	//
	// fmt.Printf("#%d : %v \"%s\"\n", i, bounds, fileName)
}

func (s *Screen) start(ctx context.Context, out chan []byte) {
	// part := time.After(time.Second)

	for {
		select {
		case <-ctx.Done():
			s.codec.Flush()
			close(out)
			return
		case <-time.After(time.Millisecond * time.Duration(1000/s.fps)):
			if img, err := s.makeScreenshot(); err == nil {
				err := s.codec.Encode(img)
				if err == nil {
					out <- s.buff.Bytes()
					s.buff.Reset()
				}
			}
		}
	}
}

func (s *Screen) makeScreenshot() (image.Image, error) {
	bounds := screenshot.GetDisplayBounds(s.id)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, err
	}

	// var b bytes.Buffer
	// w := bufio.NewWriter(&b)
	// jpeg.Encode(w, img, &jpeg.Options{Quality: 50})

	return img, err
	// return convert(img), err
}

func convert(original *image.RGBA) *image.YCbCr {
	bounds := original.Bounds()
	converted := image.NewYCbCr(bounds, image.YCbCrSubsampleRatio420)

	for row := 0; row < bounds.Max.Y; row++ {
		for col := 0; col < bounds.Max.X; col++ {
			r, g, b, _ := original.At(col, row).RGBA()
			y, cb, cr := color.RGBToYCbCr(uint8(r), uint8(g), uint8(b))

			converted.Y[converted.YOffset(col, row)] = y
			converted.Cb[converted.COffset(col, row)] = cb
			converted.Cr[converted.COffset(col, row)] = cr
		}
	}

	return converted
}

func (s *Screen) initCodec() {
	opts := &x264.Options{
		Width:     s.Bounds.Dx(),
		Height:    s.Bounds.Dy(),
		FrameRate: 10,
		Tune:      "film",
		Preset:    "ultrafast",
		Profile:   "baseline",
		LogLevel:  x264.LogError,
	}

	w := bufio.NewWriter(&s.buff)
	s.codec, _ = x264.NewEncoder(w, opts)
}
