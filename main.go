package main

import (
	"log"

	"./telegram"
)

func main() {
	log.Fatal(telegram.StartBot("./config.yaml"))
}
