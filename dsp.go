package main

import (
	"fmt"
	"github.com/racerxdl/limedrv"
	"log"
	"strings"
	"time"
)

var lastFFT = time.Now()
var gain float64
var lpf float64
var channel int
var sampleRate float64
var antenna int32
var antennaStringData = ""
var antennaList []string
var antennaCount int32
var centerFreq float64
var fftSizes = "128\x00256\x00512\x001024\x002048\x004096\x008192\x0016384"
var fftSizesLen = int32(len(strings.Split(fftSizes, "\x00")))

var selectedFFTSize = int32(5)
var dev *limedrv.LMSDevice

var dspLoaded = false
var dspLoadError = ""
var isRunning = false

func OnSamples(data []complex64, channel int, timestamp uint64) {
	if time.Since(lastFFT) > time.Second/60 {
		samplesMtx.Lock()
		defer samplesMtx.Unlock()
		fftSamples = data[:fftSize]
		go Gen()
		lastFFT = time.Now()
	}
}

func Start() {
	if dspLoaded && !isRunning {
		log.Println("Starting DSP")
		dev.Start()
		isRunning = true
	}
}

func Stop() {
	if dspLoaded && isRunning {
		log.Println("Stopping DSP")
		dev.Stop()
		isRunning = false
	}
}

func IsRunning() bool {
	return isRunning
}

func InitializeLimeSDR() {
	dspLoadError = ""
	devices := limedrv.GetDevices()

	if len(devices) == 0 {
		dspLoadError = "No devices found"
		return
	}

	dev = limedrv.Open(devices[0])
	gain = 0.4
	lpf = 10e6
	channel = 0
	sampleRate = 5e6
	antenna = 0
	centerFreq = 106.3e6

	dev.SetCallback(OnSamples)
	dev.SetSampleRate(sampleRate, 4)
	dev.RXChannels[channel].
		SetAntenna(int(antenna)).
		SetGainNormalized(gain).
		SetLPF(lpf).
		SetCenterFrequency(centerFreq).
		EnableLPF().
		Enable()

	antennaCount = int32(len(dev.RXChannels[channel].Antennas))
	antennaList = make([]string, antennaCount)

	for i := 0; i < len(dev.RXChannels[channel].Antennas); i++ {
		var ant = dev.RXChannels[channel].Antennas[i]
		antennaStringData += fmt.Sprintf("%s\x00", ant.Name)
		antennaList[i] = ant.Name
	}

	dspLoaded = true
}
