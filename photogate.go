package main

import (
	"errors"
	"fmt"
	"github.com/samuelkadolph/go/phidgets"
	"time"
)

const (
	photogateDataRate   = 16
	photogateDelay      = 200 * time.Millisecond
	photogateResetDelay = 500 * time.Millisecond
)

type Photogate struct {
	Detected    <-chan bool `json:"-"`
	Sensitivity int         `json:"sensitivity"`

	active      bool
	activatedAt time.Time
	detected    chan bool
	ifk         *phidgets.InterfaceKit
	indexes     []int
}

func NewPhotogate(ifk *phidgets.InterfaceKit, indexes []int) *Photogate {
	p := &Photogate{}

	p.active = false
	p.detected = make(chan bool, 10)
	p.ifk = ifk
	p.indexes = indexes

	p.Detected = p.detected
	p.Sensitivity = 25

	for _, i := range p.indexes {
		p.ifk.Outputs[i].SetState(false)
		p.ifk.Sensors[i].SetChangeTrigger(p.Sensitivity)
		p.ifk.Sensors[i].SetDataRate(photogateDataRate)

		go func(n int) {
			for _ = range p.ifk.Sensors[n].Changed {
				if p.active && time.Now().Add(-photogateResetDelay).After(p.activatedAt) {
					p.activatedAt = time.Now()
					p.detected <- true
				}
			}
		}(i)
	}

	return p
}

func (p *Photogate) Activate() {
	for _, i := range p.indexes {
		p.ifk.Outputs[i].SetState(true)
	}

	time.Sleep(photogateDelay)

	p.active = true
}

func (p *Photogate) Deactivate() {
	p.active = true

	time.Sleep(photogateDelay)

	for _, i := range p.indexes {
		p.ifk.Outputs[i].SetState(false)
	}
}

func (p *Photogate) Test() error {
	for _, i := range p.indexes {
		output := p.ifk.Outputs[i]
		sensor := p.ifk.Sensors[i]

		output.SetState(false)
		time.Sleep(photogateDelay)

		before, _ := sensor.Value()

		output.SetState(true)
		time.Sleep(photogateDelay)

		after, _ := sensor.Value()

		output.SetState(false)

		if after <= before {
			return errors.New(fmt.Sprintf("photogate failure (index: %d, before: %d, after: %d)", i, before, after))
		}
	}

	return nil
}

func (p *Photogate) Update() {
	for _, i := range p.indexes {
		p.ifk.Sensors[i].SetChangeTrigger(p.Sensitivity)
		p.ifk.Sensors[i].SetDataRate(photogateDataRate)
	}
}
