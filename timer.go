package main

import (
	"time"
)

type Timer struct {
	Done <-chan bool

	death   chan bool
	done    chan bool
	tick    <-chan time.Time
	timeout time.Duration
}

func NewTimer(timeout time.Duration) *Timer {
	t := &Timer{}

	t.death = make(chan bool)
	t.done = make(chan bool)
	t.tick = time.After(t.timeout)
	t.timeout = timeout

	t.Done = t.done

	go func() {
		for {
			select {
			case <-t.tick:
				select {
				case t.done <- true:
				default:
				}
			case <-t.death:
			}
		}
	}()

	return t
}

func (t *Timer) Touch() {
	t.tick = time.After(t.timeout)
	t.death <- true
}
