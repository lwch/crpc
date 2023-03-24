package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/lwch/crpc"
	"github.com/lwch/crpc/encoding/compress"
	"github.com/lwch/crpc/encoding/encrypt"
	"github.com/lwch/crpc/example/ping"
)

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cli, err := crpc.NewClient(ping.Listen)
	assert(err)
	defer cli.Close()
	cli.SetEncrypter(encrypt.New(encrypt.Aes, ping.Key))
	cli.SetCompresser(compress.New(compress.Gzip))
	for {
		req, err := http.NewRequest(http.MethodGet, "http://localhost/ping", nil)
		assert(err)
		rep, err := cli.Call(req, time.Second)
		assert(err)
		data, err := io.ReadAll(rep.Body)
		assert(err)
		log.Printf("status_code=%d, data=%s", rep.StatusCode, string(data))
		time.Sleep(time.Second)
	}
}
