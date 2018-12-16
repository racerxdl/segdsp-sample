package main

import (
	"fmt"
	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	"github.com/racerxdl/segdsp/demodcore"
	"github.com/racerxdl/segdsp/dsp"
	"github.com/racerxdl/segdsp/dsp/fft"
	"github.com/racerxdl/segdsp/tools"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"
)

// region Constants
const imgWidth = 1024
const imgHeight = 512
const fftHeight = 256
const wtfHeight = imgHeight - fftHeight
const hGridSteps = 6
const vGridSteps = 8

// endregion
// region Variables
var fftOffset = float32(-40)
var fftScale = float32(4)

var fftCache []float32
var acc = float32(4.5)
var fftSize = int32(4096)

var img = image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
var gc = draw2dimg.NewGraphicContext(img)
var fftSamples = make([]complex64, fftSize)

var samplesMtx = sync.Mutex{}

var isUpdated = false

var window = dsp.BlackmanHarris(int(fftSize), 61)

// endregion

var waterFallLut = make([]color.Color, 0)
var waterFallBuffers = make([]*image.RGBA, wtfHeight)

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

	// From GQRX: https://github.com/csete/gqrx -> qtgui/plotter.cpp
	for i := 0; i < 256; i++ {
		if i < 20 {
			// level 0: black background
			waterFallLut = append(waterFallLut, color.NRGBA{A: 255})
		} else if (i >= 20) && (i < 70) {
			// level 1: black -> blue
			waterFallLut = append(waterFallLut, color.NRGBA{B: uint8(140 * (i - 20) / 50), A: 255})
		} else if (i >= 70) && (i < 100) {
			// level 2: blue -> light-blue / greenish
			waterFallLut = append(waterFallLut, color.NRGBA{R: uint8(60 * (i - 70) / 30), G: uint8(125 * (i - 70) / 30), B: uint8(115*(i-70)/30 + 140), A: 255})
		} else if (i >= 100) && (i < 150) {
			// level 3: light blue -> yellow
			waterFallLut = append(waterFallLut, color.NRGBA{R: uint8(195*(i-100)/50 + 60), G: uint8(130*(i-100)/50 + 125), B: uint8(255 - (255 * (i - 100) / 50)), A: 255})
		} else if (i >= 150) && (i < 250) {
			// level 4: yellow -> red
			waterFallLut = append(waterFallLut, color.NRGBA{R: 255, G: uint8(255 - 255*(i-150)/100), A: 255})
		} else {
			// level 5: red -> white
			waterFallLut = append(waterFallLut, color.NRGBA{R: 255, G: uint8(255 * (i - 250) / 5), B: uint8(255 * (i - 250) / 5), A: 255})
		}
	}
}

func FrequencyToPixelX(frequency, width float64) float64 {
	var hzPerPixel = sampleRate / float64(width)
	var delta = centerFreq - float64(frequency)
	var centerX = float64(width / 2)

	return centerX + (delta / hzPerPixel)
}

func drawGrid(gc *draw2dimg.GraphicContext, img *image.RGBA, fftOffset, fftScale, width int) {
	var lineColor = color.NRGBA{R: 255, G: 127, B: 127, A: 127}
	gc.Save()
	gc.SetFontSize(10)
	gc.SetFillColor(color.NRGBA{R: 127, G: 127, B: 127, A: 255})

	var startFreq = centerFreq - (sampleRate / 2)
	var hzPerPixel = sampleRate / float64(width)

	// region Draw dB Scale Grid
	for i := 0; i < hGridSteps; i++ {
		var y = float64(i) * (float64(fftHeight) / float64(hGridSteps))
		var dB = int(float64(fftOffset) - float64(y)/float64(fftScale))
		DrawLine(0, float32(y), float32(width), float32(y), lineColor, img)
		gc.FillStringAt(fmt.Sprintf("%d dB", dB), 5, y-5)
	}
	// endregion
	// region Draw Frequency Scale Grid
	for i := 0; i < vGridSteps; i++ {
		var x = math.Round(float64(i) * (float64(width) / float64(vGridSteps)))
		DrawLine(float32(x), 0, float32(x), float32(fftHeight), lineColor, img)
		var v = startFreq + float64(x)*hzPerPixel
		v2, unit := toNotationUnit(float32(v))
		gc.FillStringAt(fmt.Sprintf("%.2f %sHz", v2, unit), x+10, float64(fftHeight)-10)
	}
	// endregion
	gc.Restore()
}

func drawChannelOverlay(gc *draw2dimg.GraphicContext, img *image.RGBA, width int) {
	if demodulator != nil {
		var dp = demodulator.GetDemodParams()
		if dp != nil {
			var demodParams = dp.(demodcore.FMDemodParams)
			var channelWidth = demodParams.SignalBandwidth

			var startX = FrequencyToPixelX(centerFreq-(channelWidth/2), float64(width))
			var endX = FrequencyToPixelX(centerFreq+channelWidth/2, float64(width))

			for i := -1; i < 1; i++ {
				DrawLine(float32(startX)+float32(i), 0, float32(startX)+float32(i), fftHeight, color.NRGBA{192, 0, 0, 255}, img)
				DrawLine(float32(endX)+float32(i), 0, float32(endX)+float32(i), fftHeight, color.NRGBA{192, 0, 0, 255}, img)
			}
		}
	}
}
func combine(c1, c2 color.Color) color.Color {
	r, g, b, a := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	return color.RGBA{
		R: uint8((r + r2) >> 9), // div by 2 followed by ">> 8"  is ">> 9"
		G: uint8((g + g2) >> 9),
		B: uint8((b + b2) >> 9),
		A: uint8((a + a2) >> 9),
	}
}

func DrawLine(x0, y0, x1, y1 float32, color color.Color, img *image.RGBA) {
	// DDA
	_, _, _, a := color.RGBA()
	needsCombine := a != 255 && a != 0
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
		if needsCombine {
			var p = img.At(int(x), int(y))
			img.Set(int(x), int(y), combine(p, color))
		} else {
			img.Set(int(x), int(y), color)
		}
		x = x + xinc
		y = y + yinc
	}
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
		var v = tools.ComplexAbsSquared(fftResult[i]) * float32(1.0/sampleRate)
		fftReal[i] = float32(10 * math.Log10(float64(v)))
		fftReal[i] = (fftCache[i]*(acc-1) + fftReal[i]) / acc
		if tools.IsNaN(fftReal[i]) {
			fftReal[i] = 0
		}
		if i > 0 {
			fftReal[i] = lastV*0.4 + fftReal[i]*0.6
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
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.NRGBA{A: 255}}, image.ZP, draw.Src)

	var lastX = float32(0)
	var lastY = float32(0)

	var maxVisible = float64(fftOffset) - float64(0)/float64(fftScale)
	var minVisible = float64(fftOffset) - float64(fftHeight)/float64(fftScale)
	var deltaVisible = math.Abs(maxVisible - minVisible)
	var scale = 256 / deltaVisible

	var wtfBuff = image.NewRGBA(image.Rect(0, 0, imgWidth, 1))

	for i := 0; i < len(fftReal); i++ {
		var iPos = (i + len(fftReal)/2) % len(fftReal)
		var s = float32(fftReal[iPos])
		var v = float32((fftOffset)-s) * float32(fftScale)
		var x = float32(i) / widthScale
		if i != 0 {
			DrawLine(lastX, lastY, x, v, color.NRGBA{R: 0, G: 127, B: 127, A: 255}, img)
		}

		waterFallZ := (float64(s) - minVisible) * scale
		if waterFallZ > 255 {
			waterFallZ = 255
		} else if waterFallZ < 0 {
			waterFallZ = 0
		}

		for i := lastX; i <= x; i++ { // For FFT Width < imgWidth
			wtfBuff.Set(int(i), 0, waterFallLut[uint8(waterFallZ)])
		}

		lastX = x
		lastY = v
	}

	if waterFallBuffers[0] != nil {
		if waterFallBuffers[0].Rect.Dx() != wtfBuff.Rect.Dx() { // Clear, wrong dimension
			waterFallBuffers = make([]*image.RGBA, wtfHeight)
		}
	}

	// Shift the waterfall forward, and add current as first.
	waterFallBuffers = append([]*image.RGBA{wtfBuff}, waterFallBuffers[:wtfHeight-1]...)

	for i := 0; i < len(waterFallBuffers); i++ {
		line := waterFallBuffers[i]
		if line != nil {
			draw.Draw(img, image.Rect(0, i+fftHeight, line.Rect.Dx(), i+1+fftHeight), line, image.ZP, draw.Src)
		}
	}

	drawGrid(gc, img, int(fftOffset), int(fftScale), imgWidth)
	drawChannelOverlay(gc, img, imgWidth)
	isUpdated = true
	// endregion
}
