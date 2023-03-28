package main

import (
	"log"

	"github.com/lwch/crpc"
	"github.com/lwch/crpc/encoding/compress"
	"github.com/lwch/crpc/encoding/encrypt"
	"github.com/lwch/crpc/example"
)

func main() {
	svr := crpc.NewServer(crpc.ServerConfig{
		Encrypter:  encrypt.New(encrypt.Aes, example.Key),
		Compresser: compress.New(compress.Gzip),
		OnAccept: func(s *crpc.Stream) {
			defer s.Close()
			buf := make([]byte, 1024)
			for {
				n, err := s.Read(buf)
				if err != nil {
					return
				}
				log.Printf("%s recved", string(buf[:n]))
				_, err = s.Write([]byte("pong"))
				if err != nil {
					return
				}
			}
		},
	})
	defer svr.Close()
	svr.ListenAndServe(example.Listen)
}
