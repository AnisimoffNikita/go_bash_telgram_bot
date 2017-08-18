package main

import (
	"fmt"
	"log"

	"github.com/AnisimoffNikita/go_bash_telgram_bot/bash"
)

func main() {

	quotes, err := bash.GetQuotes("abyss", 1)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(quotes[0].Text)

}
