package main

import (
	"flag"
	"log"

	"./telegram"
)

func main() {
	configPtr := flag.String("c", "config.yaml", "config file path")
	flag.Parse()

	log.Fatal(telegram.StartBot(*configPtr))
}
