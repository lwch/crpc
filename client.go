package crpc

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/lwch/crpc/encrypt"
	"github.com/lwch/logging"
)

var errReconnect = errors.New("reconnecting")

// Client rpc client
type Client struct {
	sync.RWMutex
	addr string
	tp   *transport
}

// NewClient create client
func NewClient(addr string, encoder *encrypt.Encoder) (*Client, error) {
	conn, err := dial(addr, 1)
	if err != nil {
		return nil, err
	}
	return &Client{
		addr: addr,
		tp:   new(conn, encoder),
	}, nil
}

func dial(addr string, retry int) (net.Conn, error) {
	for i := 0; i < retry; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			return conn, nil
		}
		time.Sleep(time.Second)
	}
	return nil, fmt.Errorf("transport: dial more than %d times", retry)
}

// Serve client serve
func (cli *Client) Serve() error {
	encoder := cli.tp.encoder
	for {
		err := cli.tp.Serve()
		if err != nil {
			logging.Error("serve %s: %v", cli.addr, err)
		}
		cli.Lock()
		cli.tp.Close()
		cli.tp = nil
		cli.Unlock()
		conn, err := dial(cli.addr, 5)
		if err != nil {
			return err
		}
		cli.Lock()
		cli.tp = new(conn, encoder)
		cli.Unlock()
	}
}

// Call call http request
func (cli *Client) Call(req *http.Request, timeout time.Duration) (*http.Response, error) {
	cli.RLock()
	tp := cli.tp
	cli.RUnlock()
	if tp == nil {
		return nil, errReconnect
	}
	return tp.Call(req, timeout)
}

// OpenStream open stream
func (cli *Client) OpenStream(timeout time.Duration) (*Stream, error) {
	cli.RLock()
	tp := cli.tp
	cli.RUnlock()
	if tp == nil {
		return nil, errReconnect
	}
	return tp.OpenStream(timeout)
}
