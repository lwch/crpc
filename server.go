package crpc

import (
	"net"

	"github.com/lwch/crpc/encrypt"
	"github.com/lwch/logging"
)

// AcceptStreamHandlerFunc handler func after accept
type AcceptStreamHandlerFunc func(*Stream)

// Server rpc server
type Server struct {
	encoder        *encrypt.Encoder
	onRequest      RequestHandlerFunc
	onAcceptStream AcceptStreamHandlerFunc
}

// NewServer create server
func NewServer(encoder *encrypt.Encoder, onRequest RequestHandlerFunc,
	onAcceptStream AcceptStreamHandlerFunc) *Server {
	return &Server{
		encoder:        encoder,
		onRequest:      onRequest,
		onAcceptStream: onAcceptStream,
	}
}

// ListenAndServe listen and serve
func (svr *Server) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go svr.handle(conn)
	}
}

func (svr *Server) handle(conn net.Conn) {
	defer conn.Close()
	tp := new(conn, svr.encoder)
	defer tp.Close()
	tp.SetOnRequest(svr.onRequest)
	go svr.acceptStream(tp)
	tp.Serve()
}

func (svr *Server) acceptStream(tp *transport) {
	for {
		stream, err := tp.AcceptStream()
		if err != nil {
			logging.Error("accept stream: %v", err)
			return
		}
		go svr.onAcceptStream(stream)
	}
}
