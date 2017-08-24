package main

import (
	"flag"
	"log"

	"./bot"
)

func main() {
	configPtr := flag.String("c", "config.yaml", "config file path")
	flag.Parse()

	log.Fatal(bot.StartBot(*configPtr))
}
