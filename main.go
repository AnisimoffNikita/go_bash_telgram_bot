package main

import (
	"log"

	"./telegram"
)

func main() {
	log.Fatal(telegram.NewBot("config.json", true))
}
