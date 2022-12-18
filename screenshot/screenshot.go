package screen

import (
	"context"
	"image"
	"image/color"
	"time"

	"github.com/kbinani/screenshot"
)

type Screen struct {
	id     int
	fps    int
	Bounds image.Rectangle
}

func InitDisplays(fps int) (result []*Screen) {
	if fps > 100 {
		fps = 100
	}

	n := screenshot.NumActiveDisplays()
	for i := 0; i < n; i++ {
		result = append(result, &Screen{id: i, fps: fps, Bounds: screenshot.GetDisplayBounds(i)})
	}

	return result
}

func (s *Screen) StartRecord(ctx context.Context) chan image.Image {
	result := make(chan image.Image)
	go s.start(ctx, result)

	return result

	// fileName := fmt.Sprintf("%d_%dx%d.jpg", i, bounds.Dx(), bounds.Dy())
	// file, _ := os.Create(fileName)
	// defer file.Close()
	// png.Encode(file, img)
	//
	// fmt.Printf("#%d : %v \"%s\"\n", i, bounds, fileName)
}

func (s *Screen) start(ctx context.Context, out chan image.Image) {
	// part := time.After(time.Second)

	for {
		select {
		case <-ctx.Done():
			close(out)
			return
		// case <-part:
		// 	return
		case <-time.After(time.Millisecond * time.Duration(1000/s.fps)):
			if img, err := s.makeScreenshot(); err == nil {
				out <- img
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
