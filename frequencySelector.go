package main

import (
	"bytes"
	"fmt"
	"github.com/golang-ui/nuklear/nk"
	"image"
	"image/draw"
	"image/png"
)

var arrowUpImage nk.Image
var arrowUpImageTex int32
var arrowDownImage nk.Image
var arrowDownImageTex int32

func InitFrequencySelectorImages() {
	arrowUpImageTex = -1
	arrowDownImageTex = -1

	imageBytes := MustAsset("assets/arrow_up.png")
	b := bytes.NewBuffer(imageBytes)
	img, err := png.Decode(b)
	if err != nil {
		panic(err)
	}
	m := image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), img, img.Bounds().Min, draw.Over)
	arrowUpImage, arrowUpImageTex = rgbaTex(arrowUpImageTex, m)

	imageBytes = MustAsset("assets/arrow_down.png")
	b = bytes.NewBuffer(imageBytes)
	img, err = png.Decode(b)
	if err != nil {
		panic(err)
	}
	m = image.NewRGBA(img.Bounds())
	draw.Draw(m, m.Bounds(), img, img.Bounds().Min, draw.Over)
	arrowDownImage, arrowDownImageTex = rgbaTex(arrowDownImageTex, m)
}

type UIFrequencySelector struct {
	frequency    uint32
	maxFrequency uint32
	minFrequency uint32
}

func MakeUIFrequencySelector(minFrequency, maxFrequency uint32) *UIFrequencySelector {
	return &UIFrequencySelector{
		frequency:    106300,
		maxFrequency: maxFrequency,
		minFrequency: minFrequency,
	}
}

func (fs *UIFrequencySelector) SetFrequency(frequency uint32) {
	fs.frequency = frequency
}

func (fs *UIFrequencySelector) GetFrequency() uint32 {
	return fs.frequency
}

func (fs *UIFrequencySelector) onUpButtonClick(n int) {
	var v = int64(PowInt(10, uint32(n)))
	var z = int64(fs.frequency) + v
	if z > int64(fs.maxFrequency) {
		fs.frequency = fs.maxFrequency
	} else {
		fs.frequency = uint32(z)
	}
}

func (fs *UIFrequencySelector) onDownButtonClick(n int) {
	var v = int64(PowInt(10, uint32(n)))
	var z = int64(fs.frequency) - v
	if z < int64(fs.minFrequency) {
		fs.frequency = fs.minFrequency
	} else {
		fs.frequency = uint32(z)
	}
}

func (fs *UIFrequencySelector) ShowAndUpdate(ctx *nk.Context) {
	var frequencyString = fmt.Sprintf("%010d", fs.frequency)
	//var mouseX, mouseY = ctx.Input().Mouse().Pos()

	// Increment Buttons
	nk.NkLayoutRowDynamic(ctx, 20, 12)
	{
		var z = 0
		for i := 0; i < 12; i++ {
			if i != 4 && i != 8 {
				if nk.NkButtonImage(ctx, arrowUpImage) > 0 {
					fs.onUpButtonClick(9 - z)
				}
				z++
			} else {
				nk.NkLabel(ctx, "", 0)
			}
		}
	}
	// Numbers
	nk.NkLayoutRowDynamic(ctx, 20, 12)
	{
		//nk.NkLabel(ctx, "HUEBR",  nk.TextLeft)
		nk.NkStyleSetFont(ctx, fonts["sans32"].Handle())
		var z = 0
		for i := 0; i < 12; i++ {
			if i == 4 || i == 8 {
				nk.NkLabel(ctx, ".", nk.TextAlignCentered|nk.TextAlignMiddle)
			} else {
				nk.NkLabel(ctx, string(frequencyString[z]), nk.TextAlignCentered|nk.TextAlignMiddle)
				z++
			}

			if nk.NkWidgetIsHovered(ctx) > 0 {

			}
		}
	}
	// Decrement Buttons
	nk.NkLayoutRowDynamic(ctx, 20, 12)
	{
		var z = 0
		for i := 0; i < 12; i++ {
			if i != 4 && i != 8 {
				if nk.NkButtonImage(ctx, arrowDownImage) > 0 {
					fs.onDownButtonClick(9 - z)
				}
				z++
			} else {
				nk.NkLabel(ctx, "", 0)
			}
		}
	}
}
