GO = go

BINDATA = $(GOPATH)/bin/go-bindata

all: horn.mp3.go fmt test

fmt:
	$(GO) fmt

horn.mp3.go: horn.mp3 | $(BINDATA)
	$(BINDATA) -f "hornMP3" -i "$<" -o "$@" -p "main"

test:
	$(GO) test

$(BINDATA):
	$(GO) get github.com/jteeuwen/go-bindata

.PHONY: all fmt test
