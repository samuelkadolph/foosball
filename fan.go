/*
 *     POWER    TIMER    OSC      SPEED    WIND                    CHECK
 *     |  |     |  |     |  |     |  |     |  |                   |      |
 * 00010000 00100000 00110000 01000011 01010000 00000000 01100000 01110011
 *
 * POWER: OFF=0 ON=1
 * TIMER: 0.0=0 0.5=1 1.0=2 1.5=3 2.0=4 2.5=5 3.0=6 3.5=7 4.0=8 4.5=9 5.0=10 5.5=11 6.0=12 6.5=13 7.0=14 7.5=15
 *   OSC: OFF=0 ON=1
 * SPEED: LOW=0 MED=1 HIGH=2 ECO=3
 *  WIND: NORMAL=0 NATURAL=1 SLEEPING=2
 * CHECK: XOR of all other bytes
 */

package main

import (
	"encoding/json"
	"github.com/samuelkadolph/go/phidgets"
)

const (
	FanOscillationOff = 0
	FanOscillationOn  = 1

	FanPowerOff = 0
	FanPowerOn  = 1

	FanSpeedLow    = 0
	FanSpeedMedium = 1
	FanSpeedHigh   = 2
	FanSpeedEco    = 3

	FanTimerOff   = 0x0
	FanTimer0h30m = 0x1
	FanTimer1h    = 0x2
	FanTimer1h30m = 0x3
	FanTimer2h    = 0x4
	FanTimer2h30m = 0x5
	FanTimer3h    = 0x6
	FanTimer3h30m = 0x7
	FanTimer4h    = 0x8
	FanTimer4h30m = 0x9
	FanTimer5h    = 0xA
	FanTimer5h30m = 0xB
	FanTimer6h    = 0xC
	FanTimer6h30m = 0xD
	FanTimer7h    = 0xE
	FanTimer7h30m = 0xF

	FanWindNormal   = 0
	FanWindNatural  = 1
	FanWindSleeping = 2
)

type FanOscillation uint8
type FanPower uint8
type FanSpeed uint8
type FanTimer uint8
type FanWind uint8

type Fan struct {
	Oscillation FanOscillation
	Power       FanPower
	Speed       FanSpeed
	Timer       FanTimer
	Wind        FanWind

	ir *phidgets.IR
}

var (
	FanPowerToJSON = map[FanPower]string{
		FanPowerOn:  "on",
		FanPowerOff: "off",
	}
	FanSpeedToJSON = map[FanSpeed]string{
		FanSpeedLow:    "low",
		FanSpeedMedium: "medium",
		FanSpeedHigh:   "high",
		FanSpeedEco:    "eco",
	}
	JSONToFanPower = map[string]FanPower{}
	JSONToFanSpeed = map[string]FanSpeed{}
)

func NewFan(ir *phidgets.IR) *Fan {
	f := &Fan{}

	f.ir = ir

	f.Oscillation = FanOscillationOff
	f.Power = FanPowerOff
	f.Speed = FanSpeedLow
	f.Timer = FanTimerOff
	f.Wind = FanWindNormal

	return f
}

func (f *Fan) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"power": FanPowerToJSON[f.Power],
		"speed": FanSpeedToJSON[f.Speed],
	})
}

func (f *Fan) SetOscillation(oscillation FanOscillation) error {
	f.Oscillation = oscillation
	return f.TransmitIR()
}

func (f *Fan) SetPower(power FanPower) error {
	f.Power = power
	return f.TransmitIR()
}

func (f *Fan) SetSpeed(speed FanSpeed) error {
	f.Speed = speed
	return f.TransmitIR()
}

func (f *Fan) SetTimer(timer FanTimer) error {
	f.Timer = timer
	return f.TransmitIR()
}

func (f *Fan) SetWind(wind FanWind) error {
	f.Wind = wind
	return f.TransmitIR()
}

func (f *Fan) TransmitIR() error {
	code := f.irCode()
	info := phidgets.IRCodeInfo{BitCount: len(code) * 8}
	return f.ir.Transmit(code, info)
}

func init() {
	for k, v := range FanPowerToJSON {
		JSONToFanPower[v] = k
	}
	for k, v := range FanSpeedToJSON {
		JSONToFanSpeed[v] = k
	}
}

func (f *Fan) irCode() []uint8 {
	code := []uint8{0x10, 0x20, 0x30, 0x40, 0x50, 0x0, 0x60, 0x0}

	code[0] |= (uint8)(f.Power & 0xF)
	code[1] |= (uint8)(f.Timer & 0xF)
	code[2] |= (uint8)(f.Oscillation & 0xF)
	code[3] |= (uint8)(f.Speed & 0xF)
	code[4] |= (uint8)(f.Wind & 0xF)

	for i := 0; i < 7; i += 1 {
		code[7] ^= code[i]
	}

	return code
}
