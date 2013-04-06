package main

import (
	"bytes"
	"log"
	"os/exec"
)

type Horn struct {
	Enabled bool `json:"enabled"`
	Volume  int  `json:"volume"`

	bytes []byte
	queue chan bool
}

func NewHorn() *Horn {
	h := &Horn{}

	h.bytes = hornMP3()
	h.queue = make(chan bool, 4)

	h.Enabled = true

	go func() {
		for {
			select {
			case <-h.queue:
				if h.Enabled {
					cmd := exec.Command("mpg123", "-q", "-")
					cmd.Stdin = bytes.NewReader(h.bytes)

					if err := cmd.Run(); err != nil {
						log.Printf("Unable to play horn - %s", err)
					}
				}
			}
		}
	}()

	return h
}

func (h *Horn) Play() {
	select {
	case h.queue <- true:
	default:
	}
}

func (h *Horn) Update() {
}
