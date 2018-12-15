package main

import (
	"bytes"
	"fmt"
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
	"github.com/racerxdl/segdsp/tools"
	"image"
	"image/draw"
	"image/png"
	"sync"
	"unsafe"
)

type UIState struct {
	bgColor nk.Color
}

var frameImg nk.Image
var frameTex int32

var fonts = make(map[string]*nk.Font)
var playButton nk.Image
var playButtonTex int32
var stopButton nk.Image
var stopButtonTex int32
var monoAtlas *nk.FontAtlas
var sansAtlas *nk.FontAtlas

var drawLock = sync.Mutex{}

func InitializeImages() {
	playButtonTex = -1
	stopButtonTex = -1

	imageBytes := MustAsset("assets/play.png")
	b := bytes.NewBuffer(imageBytes)
	img, err := png.Decode(b)
	if err != nil {
		panic(err)
	}
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), img, img.Bounds().Min, draw.Over)
	playButton, playButtonTex = rgbaTex(playButtonTex, m)

	imageBytes = MustAsset("assets/stop.png")
	b = bytes.NewBuffer(imageBytes)
	img, err = png.Decode(b)
	if err != nil {
		panic(err)
	}
	m = image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), img, img.Bounds().Min, draw.Over)
	stopButton, stopButtonTex = rgbaTex(stopButtonTex, m)
}

func InitializeFonts() {
	var freeSansBytes = MustAsset("assets/FreeSans.ttf")
	var freeMonoBytes = MustAsset("assets/FreeMono.ttf")
	sansAtlas = nk.NewFontAtlas()
	nk.NkFontStashBegin(&sansAtlas)
	for i := 8; i <= 64; i += 2 {
		var fontName = fmt.Sprintf("sans%d", i)
		fonts[fontName] = nk.NkFontAtlasAddFromBytes(sansAtlas, freeSansBytes, float32(i), nil)
	}
	nk.NkFontStashEnd()
	monoAtlas = nk.NewFontAtlas()
	nk.NkFontStashBegin(&monoAtlas)
	for i := 8; i <= 64; i += 2 {
		var fontName = fmt.Sprintf("mono%d", i)
		fonts[fontName] = nk.NkFontAtlasAddFromBytes(monoAtlas, freeMonoBytes, float32(i), nil)
	}
	nk.NkFontStashEnd()
}

func rgbaTex(tex int32, rgba *image.RGBA) (nk.Image, int32) {
	var t uint32
	if tex == -1 {
		gl.GenTextures(1, &t)
	} else {
		t = uint32(tex)
	}
	gl.BindTexture(gl.TEXTURE_2D, t)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, int32(rgba.Bounds().Dx()), int32(rgba.Bounds().Dy()),
		0, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&rgba.Pix[0]))
	gl.GenerateMipmap(gl.TEXTURE_2D)
	return nk.NkImageId(int32(t)), int32(t)
}

func buildSideMenu(win *glfw.Window, ctx *nk.Context) {
	nk.NkStyleSetFont(ctx, fonts["sans16"].Handle())
	width, height := win.GetSize()
	bounds := nk.NkRect(float32(width)-256, 0, 256, float32(height))
	update := nk.NkBegin(ctx, "Configuration", bounds, nk.WindowTitle|nk.WindowBorder)
	if update > 0 {
		nk.NkLayoutRowStatic(ctx, 32, int32(32), 1)
		{
			if IsRunning() {
				if nk.NkButtonImage(ctx, stopButton) > 0 {
					Stop()
				}
			} else {
				if nk.NkButtonImage(ctx, playButton) > 0 {
					Start()
				}
			}
		}

		nk.NkLayoutRowDynamic(ctx, 20, 1)
		{
			nk.NkLabel(ctx, fmt.Sprintf("Averaging: %f", acc), nk.TextLeft)
		}

		nk.NkLayoutRowDynamic(ctx, 20, 1)
		{
			acc = nk.NkSlideFloat(ctx, 1, acc, 16, 0.1)
		}

		nk.NkLayoutRowDynamic(ctx, 20, 1)
		{
			nk.NkLabel(ctx, fmt.Sprintf("FFT Size: %d", fftSize), nk.TextLeft)
		}
		nk.NkLayoutRowDynamic(ctx, 25, 1)
		{
			size := nk.NkVec2(nk.NkWidgetWidth(ctx), 400)
			nk.NkComboboxString(ctx, fftSizes, &selectedFFTSize, int32(fftSizesLen), 20, size)
			fftSize = int32(1 << uint32(selectedFFTSize+7))
		}

		nk.NkLayoutRowDynamic(ctx, 20, 1)
		{
			nk.NkLabel(ctx, fmt.Sprintf("Gain: %f", gain), nk.TextLeft)
		}
		nk.NkLayoutRowDynamic(ctx, 20, 1)
		{
			newGain := nk.NkSlideFloat(ctx, 0, float32(gain), 1, 0.01)
			gain = float64(newGain)
			if tools.AlmostFloatEqual(newGain, float32(gain)) && dev != nil {
				dev.RXChannels[channel].SetGainNormalized(float64(newGain))
			}
		}

		nk.NkLayoutRowDynamic(ctx, 20, 1)
		{
			nk.NkLabel(ctx, fmt.Sprintf("Antenna: %s", antennaList[antenna]), nk.TextLeft)
		}
		nk.NkLayoutRowDynamic(ctx, 25, 1)
		{
			size := nk.NkVec2(nk.NkWidgetWidth(ctx), 400)
			lastAntenna := antenna
			nk.NkComboboxString(ctx, antennaStringData, &antenna, antennaCount, 20, size)
			var isR = IsRunning()
			if lastAntenna != antenna && dev != nil {
				if isR {
					Stop()
				}
				dev.RXChannels[channel].SetAntenna(int(antenna))
				if isR {
					Start()
				}
			}
		}
	}
	nk.NkEnd(ctx)
}

func buildFFTWindow(win *glfw.Window, ctx *nk.Context) {
	width, height := win.GetSize()
	nk.NkStyleSetFont(ctx, fonts["sans16"].Handle())
	bounds := nk.NkRect(0, 0, float32(width)-256, float32(height))
	update := nk.NkBegin(ctx, "FFT Window", bounds, nk.WindowTitle)
	if update > 0 {
		if isUpdated {
			frameImg, frameTex = rgbaTex(frameTex, img)
			isUpdated = false
		}
		nk.NkLayoutRowDynamic(ctx, imgHeight, 1)
		{
			nk.NkImage(ctx, frameImg)
		}
	}
	nk.NkEnd(ctx)
}

func DrawLoading(win *glfw.Window, ctx *nk.Context) {
	ww, wh := win.GetSize()
	width := float32(600)
	height := float32(200)

	x := (float32(ww) / 2) - (width / 2)
	y := (float32(wh) / 2) - (height / 2)

	bounds := nk.NkRect(x, y, width, height)
	update := nk.NkBegin(ctx, "Loading Window", bounds, nk.WindowNoScrollbar|nk.WindowBorder)
	pad := ctx.Style().Window().Padding()

	if update > 0 {
		size := nk.NkWindowGetContentRegionSize(ctx)
		resultW := size.X() - pad.X()*2
		resultH := size.Y() - pad.Y()*2
		if dspLoadError == "" {
			nk.NkLayoutRowStatic(ctx, resultH, int32(resultW), 1)
			{
				nk.NkStyleSetFont(ctx, fonts["sans64"].Handle())
				nk.NkLabel(ctx, "Loading", nk.TextAlignCentered|nk.TextAlignMiddle)
				nk.NkStyleSetFont(ctx, fonts["sans16"].Handle())
			}
		} else {
			nk.NkLayoutRowStatic(ctx, 70, int32(resultW), 1)
			{
				nk.NkStyleSetFont(ctx, fonts["sans32"].Handle())
				nk.NkLabel(ctx, dspLoadError, nk.TextAlignCentered|nk.TextAlignMiddle)
				nk.NkStyleSetFont(ctx, fonts["sans16"].Handle())
			}
			nk.NkLayoutRowStatic(ctx, resultH-70-pad.Y()*2, int32(resultW), 1)
			{
				if nk.NkButtonLabel(ctx, "Try Again") > 0 {
					go InitializeLimeSDR()
				}
			}
		}
	}
	nk.NkEnd(ctx)
}

func gfxMain(win *glfw.Window, ctx *nk.Context, state *UIState) {
	drawLock.Lock()
	defer drawLock.Unlock()
	width, height := win.GetSize()
	nk.NkPlatformNewFrame()
	if !dspLoaded {
		DrawLoading(win, ctx)
	} else {
		buildSideMenu(win, ctx)
		buildFFTWindow(win, ctx)
	}

	// Render
	bg := make([]float32, 4)
	nk.NkColorFv(bg, state.bgColor)
	gl.Viewport(0, 0, int32(width), int32(height))
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.ClearColor(bg[0], bg[1], bg[2], bg[3])
	nk.NkPlatformRender(nk.AntiAliasingOn, maxVertexBuffer, maxElementBuffer)
	win.SwapBuffers()
}
