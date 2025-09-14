package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	carrierWidth   = 200
	carrierHeight  = 100
	carrierPadding = 20 * 1024 // 20KB
)

func createCarrierPNG(text string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, carrierWidth, carrierHeight))
	bgColor := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	textColor := color.RGBA{R: 50, G: 50, B: 50, A: 255}

	// Fill background
	for x := 0; x < carrierWidth; x++ {
		for y := 0; y < carrierHeight; y++ {
			img.Set(x, y, bgColor)
		}
	}

	// Add text
	point := fixed.Point26_6{X: fixed.Int26_6(10 * 64), Y: fixed.Int26_6(50 * 64)}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)

	// Encode to PNG
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		log.Println("Failed to encode png:", err)
		return nil, err
	}

	// Pad to 20KB
	pngData := buf.Bytes()
	if len(pngData) < carrierPadding {
		padding := make([]byte, carrierPadding-len(pngData))
		pngData = append(pngData, padding...)
	}

	return pngData, nil
}
