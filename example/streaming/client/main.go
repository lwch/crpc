package main

import (
	"context"
	"log"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	s, err := cli.OpenStream(ctx)
	assert(err)
	defer s.Close()
	buf := make([]byte, 1024)
	for {
		_, err := s.Write([]byte("ping"))
		if err != nil {
			return
		}
		n, err := s.Read(buf)
		if err != nil {
			return
		}
		log.Println(string(buf[:n]))
		time.Sleep(time.Second)
	}
}
