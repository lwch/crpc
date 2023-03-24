package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/lwch/crpc"
	"github.com/lwch/crpc/encoding/compress"
	"github.com/lwch/crpc/encoding/encrypt"
	"github.com/lwch/crpc/example/ping"
)

func main() {
	svr := crpc.NewServer(crpc.ServerConfig{
		Encrypter:  encrypt.New(encrypt.Aes, ping.Key),
		Compresser: compress.New(compress.Gzip),
		OnRequest: func(r *http.Request) (*http.Response, error) {
			log.Println("ping recved")
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("pong")),
			}, nil
		},
	})
	defer svr.Close()
	svr.ListenAndServe(ping.Listen)
}
