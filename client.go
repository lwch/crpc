package crpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/lwch/crpc/encrypt"
	"github.com/lwch/logging"
)

// ErrReconnecting reconnecting error
var ErrReconnecting = errors.New("reconnecting")

// ErrClosed closed error
var ErrClosed = errors.New("closed")

// Client rpc client
type Client struct {
	sync.RWMutex
	addr string
	tp   *transport
	// runtime
	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient create client
func NewClient(addr string, encoder *encrypt.Encoder) (*Client, error) {
	conn, err := dial(addr, 1)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	cli := &Client{
		addr:   addr,
		tp:     new(conn, encoder),
		ctx:    ctx,
		cancel: cancel,
	}
	go cli.serve()
	return cli, nil
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

// Close close client
func (cli *Client) Close() error {
	err := cli.tp.Close()
	cli.cancel()
	return err
}

func (cli *Client) serve() error {
	defer cli.cancel()
	encoder := cli.tp.encoder
	for {
		select {
		case <-cli.ctx.Done():
			return ErrClosed
		default:
		}
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
			continue
		}
		cli.Lock()
		cli.tp = new(conn, encoder)
		cli.Unlock()
	}
}

// Call call http request
func (cli *Client) Call(req *http.Request, timeout time.Duration) (*http.Response, error) {
	select {
	case <-cli.ctx.Done():
		return nil, ErrClosed
	default:
	}
	cli.RLock()
	tp := cli.tp
	cli.RUnlock()
	if tp == nil {
		return nil, ErrReconnecting
	}
	return tp.Call(req, timeout)
}

// OpenStream open stream
func (cli *Client) OpenStream(timeout time.Duration) (*Stream, error) {
	select {
	case <-cli.ctx.Done():
		return nil, ErrClosed
	default:
	}
	cli.RLock()
	tp := cli.tp
	cli.RUnlock()
	if tp == nil {
		return nil, ErrReconnecting
	}
	return tp.OpenStream(timeout)
}
