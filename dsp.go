package main

import (
	"fmt"
	"github.com/gordonklaus/portaudio"
	"github.com/myriadrf/limedrv"
	"github.com/racerxdl/go.fifo"
	"github.com/racerxdl/segdsp/demodcore"
	"github.com/racerxdl/segdsp/dsp"
	"log"
	"strings"
	"time"
)

const audioBufferSize = 8192

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

var dspLoaded TAtomBool
var dspLoadError = ""
var isRunning = false

var dcFilter = dsp.MakeDCFilter()
var demodulator demodcore.DemodCore

var audioStream *portaudio.Stream
var audioFifo = fifo.NewQueue()

func OnSamples(data []complex64, _ int, _ uint64) {
	go DoDemod(data)
	go DoFFT(data)
}

func DoFFT(data []complex64) {
	samplesMtx.Lock()
	defer samplesMtx.Unlock()
	if time.Since(lastFFT) > time.Second/60 {
		fftLock.Lock()
		fftSamples = data[:fftSize]
		fftLock.Unlock()
		data = dcFilter.Work(fftSamples)
		go Gen()
		UpdateVisuals()
		lastFFT = time.Now()
	}
}

func DoDemod(samples []complex64) {
	samplesMtx.Lock()
	defer samplesMtx.Unlock()
	out := demodulator.Work(samples)
	if out != nil {
		var o = out.(demodcore.DemodData)
		var nBf = make([]float32, len(o.Data))
		copy(nBf, o.Data)
		var buffs = len(nBf) / audioBufferSize
		for i := 0; i < buffs; i++ {
			audioFifo.Add(nBf[8192*i : 8192*(i+1)])
		}
	}
}

func UpdateVisuals() {

}

func Start() {
	if dspLoaded.Get() && !isRunning {
		log.Println("Starting DSP")
		dev.Start()
		err := audioStream.Start()
		if err != nil {
			panic(err)
		}
		isRunning = true
	}
}

func Stop() {
	if dspLoaded.Get() && isRunning {
		log.Println("Stopping DSP")
		dev.Stop()
		log.Println("Stopping Audio")
		err := audioStream.Stop()
		if err != nil {
			log.Println("Error stopping audio: ", err)
		}
		isRunning = false
		log.Println("Stopped")
	}
}

func IsRunning() bool {
	return isRunning
}

func ProcessAudio(out []float32) {
	if audioFifo.Len() > 0 {
		var z = audioFifo.Next().([]float32)
		copy(out, z)
	} else {
		for i := range out {
			out[i] = 0
		}
	}
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
	sampleRate = 2e6
	antenna = 0
	centerFreq = 96.9e6 // 106.3e6

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

	frequencySelector.SetFrequency(uint32(centerFreq))

	for i := 0; i < len(dev.RXChannels[channel].Antennas); i++ {
		var ant = dev.RXChannels[channel].Antennas[i]
		antennaStringData += fmt.Sprintf("%s\x00", ant.Name)
		antennaList[i] = ant.Name
	}

	demodulator = demodcore.MakeWBFMDemodulator(uint32(sampleRate), 192e3, 48000)

	err := portaudio.Initialize()
	if err != nil {
		dspLoadError = err.Error()
		return
	}

	h, err := portaudio.DefaultHostApi()

	if err != nil {
		dspLoadError = err.Error()
		return
	}

	//log.Printf("Audio Device: %s\n", h.DefaultOutputDevice.Name)
	//
	//for i := 0; i < len(h.Devices); i++ {
	//	log.Printf("%d: %s\n", i, h.Devices[i].Name)
	//}

	p := portaudio.HighLatencyParameters(nil, h.DefaultOutputDevice)
	p.Input.Channels = 0
	p.Output.Channels = 1
	p.SampleRate = 48000
	p.FramesPerBuffer = audioBufferSize

	// Add few empty buffers to keep up on start
	audioFifo.Add(make([]float32, audioBufferSize))
	audioFifo.Add(make([]float32, audioBufferSize))
	audioFifo.Add(make([]float32, audioBufferSize))
	audioFifo.Add(make([]float32, audioBufferSize))

	audioStream, err = portaudio.OpenStream(p, ProcessAudio)

	if err != nil {
		dspLoadError = err.Error()
		return
	}

	dspLoaded.Set(true)
}

func onDspClose() {
	if audioStream != nil {
		err := audioStream.Close()
		if err != nil {
			log.Printf("Error closing stream: %s", err)
		}
	}
	err := portaudio.Terminate()
	if err != nil {
		log.Printf("Error terminating portaudio: %s", err)
	}
}
