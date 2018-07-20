package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mylanconnolly/srv"
)

type client chan<- string

var (
	clients  = make(map[client]struct{})
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string)
)

func main() {
	s, err := srv.NewServer(srv.ProtocolTCP, "127.0.0.1:1337")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	s.Log = true

	s.AddStreamingEndpoint("message", func(meta srv.Metadata, client *srv.Client) error {
		ch := make(chan string)

		go func() {
			for msg := range ch {
				fmt.Fprintln(client, msg)
			}
		}()

		who := client.RemoteAddr().String()

		ch <- "You are " + who
		messages <- who + " has arrived"
		entering <- ch

		input := bufio.NewScanner(client)

		for input.Scan() {
			messages <- who + ": " + input.Text()
		}
		leaving <- ch
		messages <- who + " has left"

		return client.Close()
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
		case msg := <-messages:
			for cli := range clients {
				cli <- msg
			}
		case cli := <-entering:
			clients[cli] = struct{}{}
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		case sig := <-quit:
			log.Println("Caught signal", sig, "shutting down...")
			s.Shutdown()
			return
		}
	}
}
