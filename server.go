package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"sync"
)

const (
	server = 1
)

type Server struct {
	Scored chan<- string

	fans       map[string]*Fan
	horn       *Horn
	http       *http.Server
	mu         sync.RWMutex
	photogates map[string]*Photogate
	router     *mux.Router
	scored     chan string
	streamers  []*bufio.ReadWriter
}

type hash map[string]interface{}

func NewServer(horn *Horn, fans map[string]*Fan, photogates map[string]*Photogate) *Server {
	s := &Server{}

	s.fans = fans
	s.horn = horn
	s.http = &http.Server{Handler: s}
	s.photogates = photogates
	s.router = mux.NewRouter()
	s.scored = make(chan string, 100)
	s.streamers = make([]*bufio.ReadWriter, 0)

	s.Scored = s.scored

	s.router.NewRoute().Methods("GET").Path("/fans/{name}").Handler(http.HandlerFunc(showFan))
	s.router.NewRoute().Methods("PUT").Path("/fans/{name}").Handler(http.HandlerFunc(updateFan))
	s.router.NewRoute().Methods("GET").Path("/horn").Handler(http.HandlerFunc(showHorn))
	s.router.NewRoute().Methods("PUT").Path("/horn").Handler(http.HandlerFunc(updateHorn))
	s.router.NewRoute().Methods("POST").Path("/horn/play").Handler(http.HandlerFunc(playHorn))
	s.router.NewRoute().Methods("GET").Path("/photogates/{name}").Handler(http.HandlerFunc(showPhotogate))
	s.router.NewRoute().Methods("PUT").Path("/photogates/{name}").Handler(http.HandlerFunc(updatePhotogate))
	s.router.NewRoute().Methods("GET").Path("/stream").Handler(http.HandlerFunc(stream))
	s.router.NewRoute().Methods("GET").Path("/").Handler(http.HandlerFunc(root))
	s.router.NewRoute().Handler(http.HandlerFunc(notFound))

	go s.sendMessages()

	return s
}

func (s *Server) Listen(address string) (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp4", address)
	if err != nil {
		return nil, err
	}

	l, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		return nil, err
	}

	go s.http.Serve(l)

	return l, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	context.Set(r, server, s)
	s.router.ServeHTTP(w, r)
	context.Clear(r)
}

func showFan(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)
	v := mux.Vars(r)
	f := s.fans[v["name"]]

	if f == nil {
		notFound(w, r)
		return
	}

	response(w, 200, f)
}

func updateFan(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)
	v := mux.Vars(r)
	f := s.fans[v["name"]]

	if f == nil {
		notFound(w, r)
		return
	}

	e := make(map[string]string)
	p := f.Power
	d := f.Speed

	if t := r.Form.Get("power"); t != "" {
		if u, ok := JSONToFanPower[t]; ok {
			e["power"] = "is not valid"
		} else {
			p = u
		}
	}

	if t := r.Form.Get("speed"); t != "" {
		if u, ok := JSONToFanSpeed[t]; ok {
			e["speed"] = "is not valid"
		} else {
			d = u
		}
	}

	if len(e) != 0 {
		response(w, 422, hash{"errors": e})
	} else {
		f.Power = p
		f.Speed = d

		if err := f.TransmitIR(); err != nil {
			response(w, 200, hash{"success": false, "error": err.Error()})
		} else {
			response(w, 200, hash{"success": true})
		}
	}
}

func showHorn(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)

	response(w, 200, s.horn)
}

func updateHorn(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)
	h := s.horn

	if h == nil {
		notFound(w, r)
		return
	}

	e := make(map[string]string)
	n := h.Enabled
	v := h.Volume

	if t := r.Form.Get("enabled"); t != "" {
		if u, err := strconv.ParseBool(t); err != nil {
			e["enabled"] = "is not a boolean"
		} else {
			n = u
		}
	}

	if t := r.Form.Get("volume"); t != "" {
		if u, err := strconv.Atoi(t); err != nil {
			e["volume"] = "is not a number"
		} else if 0 < u || u <= 100 {
			e["volume"] = "must be between 0 and 100"
		} else {
			v = u
		}
	}

	if len(e) != 0 {
		response(w, 422, hash{"errors": e})
	} else {
		h.Enabled = n
		h.Volume = v
		h.Update()

		response(w, 200, hash{"success": true})
	}
}

func playHorn(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)

	s.horn.Play()

	response(w, 200, hash{"success": true})
}

func showPhotogate(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)
	v := mux.Vars(r)
	p := s.photogates[v["name"]]

	if p == nil {
		notFound(w, r)
		return
	}

	response(w, 200, p)
}

func updatePhotogate(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)
	v := mux.Vars(r)
	p := s.photogates[v["name"]]

	if p == nil {
		notFound(w, r)
		return
	}

	e := make(map[string]string)
	n := p.Sensitivity

	if t := r.Form.Get("sensitivity"); t != "" {
		if u, err := strconv.Atoi(t); err != nil {
			e["sensitivity"] = "is not a number"
		} else {
			n = u
		}
	}

	if len(e) != 0 {
		response(w, 422, hash{"errors": e})
	} else {
		p.Sensitivity = n
		p.Update()

		response(w, 200, hash{"success": true})
	}
}

func stream(w http.ResponseWriter, r *http.Request) {
	s := context.Get(r, server).(*Server)

	h, ok := w.(http.Hijacker)
	if !ok {
		response(w, 500, hash{"error": "conection does not support hijacking"})
		return
	}

	_, b, err := h.Hijack()
	if err != nil {
		response(w, 500, hash{"error": err.Error()})
		return
	}

	s.mu.Lock()
	s.streamers = append(s.streamers, b)
	s.mu.Unlock()

	b.WriteString("HTTP/1.1 200 OK\r\n")
	b.WriteString("Content-Type: application/json\r\n")
	b.WriteString("Transfer-Encoding: chunked\r\n\r\n")
	b.Flush()
}

func root(w http.ResponseWriter, r *http.Request) {
	response(w, 200, hash{"hi": true})
}

func notFound(w http.ResponseWriter, r *http.Request) {
	response(w, 404, hash{"error": "not found"})
}

func response(w http.ResponseWriter, s int, b interface{}) {
	o, _ := json.Marshal(b)

	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(o)))
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(s)
	w.Write(o)
}

func (s *Server) sendMessage(i int, w *bufio.ReadWriter, b []byte) {
	c := httputil.NewChunkedWriter(w)

	_, err := c.Write(b)
	if err == nil {
		err = w.Flush()
	}
	if err != nil {
		s.mu.Lock()
		if len(s.streamers) == 1 {
			s.streamers = s.streamers[:0]
		} else {
			s.streamers[i] = s.streamers[len(s.streamers)-1]
			s.streamers = s.streamers[:len(s.streamers)-1]
		}
		s.mu.Unlock()
	}
}

func (s *Server) sendMessages() {
	type event struct {
		Type string `json:"type"`
		Team string `json:"team"`
	}

	for t := range s.scored {
		b, _ := json.Marshal(event{"scored", t})
		b = []byte(fmt.Sprintf("%s\r\n", string(b)))
		s.mu.RLock()
		for i, w := range s.streamers {
			go s.sendMessage(i, w, b)
		}
		s.mu.RUnlock()
	}
}
