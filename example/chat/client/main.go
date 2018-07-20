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

func main() {
	client, err := srv.NewClient(srv.ProtocolTCP, "127.0.0.1:1337")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if _, err = client.WriteMeta(srv.Metadata{
		Endpoint:     "message",
		EndpointType: srv.EndpointStream,
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			client.Write(append(scanner.Bytes(), '\n'))
		}
	}()

	go func() {
		scanner := bufio.NewScanner(client)

		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	for {
		select {
		case sig := <-quit:
			log.Println("Caught signal", sig, "quitting...")
			return
		}
	}
}
