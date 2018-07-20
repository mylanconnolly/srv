package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
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

	// Simple endpoint that echos back input
	s.AddRequestEndpoint("echo", func(meta srv.Metadata, w io.Writer, r io.Reader) error {
		io.Copy(w, r)
		return nil
	})

	// Simple endpoint that returns the uppercase of input
	s.AddRequestEndpoint("upper", func(meta srv.Metadata, w io.Writer, r io.Reader) error {
		buf := &bytes.Buffer{}
		io.Copy(buf, r)
		str := strings.ToUpper(buf.String())
		w.Write([]byte(str))
		return nil
	})

	// Simple endpoint that returns the lowercase of input
	s.AddRequestEndpoint("lower", func(meta srv.Metadata, w io.Writer, r io.Reader) error {
		buf := &bytes.Buffer{}
		io.Copy(buf, r)
		str := strings.ToLower(buf.String())
		w.Write([]byte(str))
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
