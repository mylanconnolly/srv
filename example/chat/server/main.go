package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mylanconnolly/srv"
)

func main() {
	s, err := srv.NewServer(srv.ProtocolTCP, "127.0.0.1:1337")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	s.Log = true

	s.AddStreamingEndpoint("message", func(meta srv.Metadata, client *srv.Client) error {
		return nil
	})

	go func() {
		if err := s.Listen(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case sig := <-quit:
			log.Println("Caught signal", sig, "shutting down...")
			s.Shutdown()
			return
		}
	}
}
