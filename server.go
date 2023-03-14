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
	listener       net.Listener
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
	var err error
	svr.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer svr.listener.Close()
	for {
		conn, err := svr.listener.Accept()
		if err != nil {
			continue
		}
		go svr.handle(conn)
	}
}

// Close close server
func (svr *Server) Close() error {
	return svr.listener.Close()
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