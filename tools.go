package main

import "sync/atomic"

var subMultiples = []string{"", "m", "µ", "n", "p", "f", "a", "z", "y"}
var multiples = []string{"", "k", "M", "G", "T", "P", "E", "Z", "Y"}

func toNotationUnit(v float32) (float32, string) {
	var unit string
	var counter = 0
	var value = v

	if value < 1 {
		for value < 1 {
			counter++
			value = value * 1e3
			if counter == 8 {
				break
			}
		}
		unit = subMultiples[counter]
	} else {
		for value > 1000 {
			counter++
			value = value / 1e3
			if counter == 8 {
				break
			}
		}
		unit = multiples[counter]
	}

	value = float32(int(value*1e2)) / 1e2
	return value, unit
}

func PowInt(a, b uint32) uint32 {
	var s = uint32(1)

	for i := uint32(0); i < b; i++ {
		s *= a
	}

	return s
}

type TAtomBool struct{ flag int32 }

func (b *TAtomBool) Set(value bool) {
	var i int32 = 0
	if value {
		i = 1
	}
	atomic.StoreInt32(&(b.flag), int32(i))
}

func (b *TAtomBool) Get() bool {
	if atomic.LoadInt32(&(b.flag)) != 0 {
		return true
	}
	return false
}
