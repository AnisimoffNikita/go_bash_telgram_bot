package main

import (
	"log"

	"github.com/AnisimoffNikita/go_bash_telgram_bot/telegram"
)

func main() {
	log.Fatal(telegram.StartBot("./config.json"))
}
