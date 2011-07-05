include $(GOROOT)/src/Make.inc

TARG = pusher
GOFILES = common.go channel.go acceptor.go pusher.go
	
include $(GOROOT)/src/Make.pkg

.PHONY: gofmt
gofmt:
	gofmt -w $(GOFILES)
