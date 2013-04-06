package main

import (
	"flag"
	"fmt"
	"github.com/samuelkadolph/go/phidgets"
	"log"
	"time"
)

const (
	AttachmentTimeout = 2 * time.Second
	FanTimeout        = 10 * time.Minute
)

var (
	host = flag.String("host", "", "Host for the server to listen on")
	port = flag.Int("port", 5555, "Port for the server to listen on")
)

func check(err error, m string, a ...interface{}) {
	if err != nil {
		log.Fatalf(m+": %s", append(a, err)...)
	}
}

func main() {
	flag.Parse()

	ifk, err := phidgets.NewInterfaceKit()
	check(err, "Unable to create InterfaceKit")
	ir1, err := phidgets.NewIR()
	check(err, "Unable to create IR")
	ir2, err := phidgets.NewIR()
	check(err, "Unable to create IR")

	check(ifk.Open(phidgets.Label{"foosifk"}), "Unable to open foosifk")
	check(ifk.WaitForAttachment(AttachmentTimeout), "Unable to attach to foosifk")
	defer ifk.Close()
	log.Printf("Attached to foosifk")
	// check(ir1.Open(phidgets.Label{"foosfan1"}), "Unable to open foosfan1")
	// check(ir1.WaitForAttachment(AttachmentTimeout), "Unable to attach to foosfan1")
	// defer ir1.Close()
	// log.Printf("Attached to foosfan1")
	check(ir2.Open(phidgets.Label{"foosfan2"}), "Unable to open foosfan2")
	check(ir2.WaitForAttachment(AttachmentTimeout), "Unable to attach to foosfan2")
	defer ir2.Close()
	log.Printf("Attached to foosfan2")

	horn := NewHorn()
	blackFan := NewFan(ir1)
	blackPhotogate := NewPhotogate(ifk, []int{0, 1, 2})
	yellowFan := NewFan(ir2)
	yellowPhotogate := NewPhotogate(ifk, []int{3, 4, 5})

	check(blackPhotogate.Test(), "Black Photogate is bad")
	log.Printf("Black Photogate is good")
	check(yellowPhotogate.Test(), "Yellow Photogate is bad")
	log.Printf("Yellow Photogate is good")

	blackPhotogate.Activate()
	defer blackPhotogate.Deactivate()
	log.Printf("Black Photogate activated")
	yellowPhotogate.Activate()
	defer yellowPhotogate.Deactivate()
	log.Printf("Yellow Photogate activated")

	fans := map[string]*Fan{
		"black":  blackFan,
		"yellow": yellowFan,
	}
	photogates := map[string]*Photogate{
		"black":  blackPhotogate,
		"yellow": yellowPhotogate,
	}
	server := NewServer(horn, fans, photogates)
	addr := fmt.Sprintf("%s:%d", *host, *port)
	listener, err := server.Listen(addr)
	check(err, "Unable to listen on %s", addr)
	defer listener.Close()
	log.Printf("Listening on %s", addr)

	fanTimer := NewTimer(FanTimeout)
	go func() {
		for {
			<-fanTimer.Done
			log.Printf("Turning fans off")
			blackFan.SetPower(FanPowerOff)
			yellowFan.SetPower(FanPowerOff)
		}
	}()

	go func() {
		for _ = range blackPhotogate.Detected {
			server.Scored <- "black"
			log.Printf("Black Scored")
			horn.Play()
			fanTimer.Touch()
		}
	}()
	go func() {
		for _ = range yellowPhotogate.Detected {
			server.Scored <- "yellow"
			log.Printf("Yellow Scored")
			horn.Play()
			fanTimer.Touch()
		}
	}()

	select {}
}
