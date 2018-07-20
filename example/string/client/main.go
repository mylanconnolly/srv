package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mylanconnolly/srv"
)

func main() {
	client, err := srv.NewClient(srv.ProtocolTCP, "localhost:1337")

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	for i := 0; i < 1000; i++ {
		statement := "Hello " + strconv.Itoa(i) + " times"

		if _, err = client.WriteDataString("echo", statement); err != nil {
			fmt.Println("Could not write to echo handler:", err)
			os.Exit(1)
		}
		_, echoStr, err := client.ReadDataString()

		if err != nil {
			fmt.Println("Could not read from echo handler:", err)
			os.Exit(1)
		}
		if _, err = client.WriteDataString("upper", statement); err != nil {
			fmt.Println("Could not write to upper handler:", err)
			os.Exit(1)
		}
		_, upperStr, err := client.ReadDataString()

		if err != nil {
			fmt.Println("Could not read from upper handler:", err)
			os.Exit(1)
		}
		if _, err = client.WriteDataString("lower", statement); err != nil {
			fmt.Println("Could not write to lower handler:", err)
			os.Exit(1)
		}
		_, lowerStr, err := client.ReadDataString()

		if err != nil {
			fmt.Println("Could not read from lower handler:", err)
			os.Exit(1)
		}
		fmt.Printf("%s\t%s\t%s\n", strings.TrimSpace(echoStr), strings.TrimSpace(lowerStr), strings.TrimSpace(upperStr))
	}
}
