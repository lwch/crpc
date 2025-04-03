package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/lwch/crpc"
	"github.com/lwch/crpc/encoding/compress"
	"github.com/lwch/crpc/encoding/encrypt"
	"github.com/lwch/crpc/example"
)

func assert(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cli, err := crpc.NewClient(example.Listen)
	assert(err)
	defer cli.Close()
	cli.SetEncrypter(encrypt.New(encrypt.Aes, example.Key))
	cli.SetCompresser(compress.New(compress.Gzip))
	for {
		func() {
			req, err := http.NewRequest(http.MethodGet, "http://localhost/ping", nil)
			assert(err)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			rep, err := cli.Call(ctx, req)
			assert(err)
			data, err := io.ReadAll(rep.Body)
			assert(err)
			log.Printf("status_code=%d, data=%s", rep.StatusCode, string(data))
		}()
		time.Sleep(time.Second)
	}
}
