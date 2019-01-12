package main

import (
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/golang-ui/nuklear/nk"
	//"github.com/pkg/profile"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	winWidth  = 1280
	winHeight = 620

	maxVertexBuffer  = 512 * 1024
	maxElementBuffer = 128 * 1024
)

func init() {
	frameTex = -1
	for i := 0; i < len(fftSamples); i++ {
		fftSamples[i] = complex64(0)
	}
	dspLoaded.Set(false)
}

func main() {
	//defer profile.Start().Stop()
	runtime.LockOSThread()
	if err := glfw.Init(); err != nil {
		log.Fatalln(err)
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 2)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	win, err := glfw.CreateWindow(winWidth, winHeight, "SegDSP Sample Application", nil, nil)
	if err != nil {
		log.Fatalln(err)
	}
	win.MakeContextCurrent()

	width, height := win.GetSize()

	if err := gl.Init(); err != nil {
		log.Fatalf("opengl initialization failed: %s", err)
	}
	gl.Viewport(0, 0, int32(width), int32(height))

	ctx := nk.NkPlatformInit(win, nk.PlatformInstallCallbacks)

	InitializeFonts()
	InitializeImages()

	nk.NkStyleSetFont(ctx, fonts["sans16"].Handle())

	exitC := make(chan os.Signal, 1)
	doneC := make(chan struct{}, 1)

	signal.Notify(exitC, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-exitC
		log.Println("Got SIGTERM!")
		go Stop()
		onDspClose()
		win.SetShouldClose(true)
		<-doneC
	}()

	state := &UIState{
		bgColor: nk.NkRgba(28, 48, 62, 255),
	}

	fpsTicker := time.NewTicker(time.Second / 60)

	Gen()
	frameImg, frameTex = rgbaTex(frameTex, img)
	go InitializeLimeSDR()

	for {
		select {
		case <-exitC:
			nk.NkPlatformShutdown()
			glfw.Terminate()
			fpsTicker.Stop()
			close(doneC)
			return
		case <-fpsTicker.C:
			if win.ShouldClose() {
				close(exitC)
				continue
			}
			glfw.PollEvents()
			gfxMain(win, ctx, state)
		}
	}
}
