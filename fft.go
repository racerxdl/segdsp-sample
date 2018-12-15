package main

import (
	"fmt"
	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/racerxdl/segdsp/dsp"
	"github.com/racerxdl/segdsp/dsp/fft"
	"github.com/racerxdl/segdsp/tools"
	"image"
	"image/color"
	"math"
	"sync"
)

const imgWidth = 1024
const imgHeight = 512
const gridSteps = 8
const fftOffset = -20
const fftScale = 4

var fftCache []float32
var acc = float32(4.5)
var fftSize = int32(4096)

var img = image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
var gc = draw2dimg.NewGraphicContext(img)
var fftSamples = make([]complex64, fftSize)

var samplesMtx = sync.Mutex{}

var isUpdated = false

var window = dsp.BlackmanHarris(int(fftSize), 61)

func drawGrid(gc *draw2dimg.GraphicContext, fftOffset, fftScale, width int) {
	gc.Save()
	gc.SetLineWidth(1)
	gc.SetFontSize(10)
	gc.SetStrokeColor(color.RGBA{R: 127, G: 127, B: 127, A: 255})
	gc.SetFillColor(color.RGBA{R: 127, G: 127, B: 127, A: 255})
	// region Draw dB Scale Grid
	for i := 0; i < gridSteps; i++ {
		var y = float64(i) * (float64(imgHeight) / float64(gridSteps))
		var dB = int(float64(fftOffset) - float64(y)/float64(fftScale))
		gc.MoveTo(0, y)
		gc.LineTo(float64(width), y)
		gc.FillStringAt(fmt.Sprintf("%d dB", dB), 5, y-5)
	}
	// endregion
	// region Draw Frequency Scale Grid

	// endregion
	gc.Stroke()
	gc.Restore()
}

func DrawLine(x0, y0, x1, y1 float32, color color.Color, img *image.RGBA) {
	// DDA
	var dx = x1 - x0
	var dy = y1 - y0
	var steps float32
	if tools.Abs(dx) > tools.Abs(dy) {
		steps = tools.Abs(dx)
	} else {
		steps = tools.Abs(dy)
	}

	var xinc = dx / steps
	var yinc = dy / steps

	var x = x0
	var y = y0
	for i := 0; i < int(steps); i++ {
		img.Set(int(x), int(y), color)
		x = x + xinc
		y = y + yinc
	}
}


func init() {
	fontBytes := MustAsset("assets/FreeSans.ttf")
	loadedFont, err := truetype.Parse(fontBytes)
	if err != nil {
		panic(err)
	}
	draw2d.RegisterFont(draw2d.FontData{
		Name:   "FreeMono",
		Family: draw2d.FontFamilyMono,
		Style:  draw2d.FontStyleNormal,
	}, loadedFont)

	gc.SetFontData(draw2d.FontData{
		Name:   "FreeMono",
		Family: draw2d.FontFamilyMono,
		Style:  draw2d.FontStyleNormal,
	})
	gc.SetLineWidth(2)
	gc.SetStrokeColor(color.RGBA{R: 255, A: 255})
	gc.SetFillColor(color.RGBA{R: 0, A: 255})
}

func Gen() {
	// region Compute FFT
	samplesMtx.Lock()
	localSamples := make([]complex64, fftSize)
	copy(localSamples, fftSamples)

	if len(window) != len(localSamples) {
		window = dsp.BlackmanHarris(int(fftSize), 61)
	}
	samplesMtx.Unlock()

	for j := 0; j < int(fftSize); j++ {
		var s = localSamples[j]
		var r = real(s) * float32(window[j])
		var i = imag(s) * float32(window[j])
		localSamples[j] = complex(r, i)
	}

	fftResult := fft.FFT(fftSamples)

	fftReal := make([]float32, len(fftResult))
	if fftCache == nil || len(fftCache) != len(fftReal) {
		fftCache = make([]float32, len(fftReal))
		for i := 0; i < len(fftReal); i++ {
			fftCache[i] = 0
		}
	}

	var lastV = float32(0)
	for i := 0; i < len(fftResult); i++ {
		// Convert FFT to Power in dB
		var v = tools.ComplexAbsSquared(fftResult[i]) * float32(1.0 / sampleRate)
		fftReal[i] = float32(10 * math.Log10(float64(v)))
		fftReal[i] = (fftCache[i] * (acc - 1) + fftReal[i]) / acc
		if tools.IsNaN(fftReal[i]) {
			fftReal[i] = 0
		}
		if i > 0 {
			fftReal[i] = lastV * 0.4 + fftReal[i] * 0.6
		}
		lastV = fftReal[i]
		fftCache[i] = fftReal[i]
	}

	// endregion
	// region Draw Image and Save
	widthScale := float32(len(fftReal)) / float32(imgWidth)

	drawLock.Lock()
	defer drawLock.Unlock()
	gc.Clear()
	gc.SetFontSize(32)
	gc.FillStringAt("FFT", 10, 10)

	var lastX = float32(0)
	var lastY = float32(0)

	for i := 0; i < len(fftReal); i++ {
		var iPos = (i + len(fftReal)/2) % len(fftReal)
		var s = float32(fftReal[iPos])
		var v = float32((fftOffset) - s) * float32(fftScale)
		var x = float32(i) / widthScale
		if i != 0 {
			DrawLine(lastX, lastY, x, v, color.RGBA{R:0, G:127, B:127, A: 255}, img)
		}
		lastX = x
		lastY = v
	}

	drawGrid(gc, fftOffset, fftScale, imgWidth)
	isUpdated = true
	// endregion
}
