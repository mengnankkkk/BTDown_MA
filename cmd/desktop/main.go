package main

import (
	"log"

	"BTDown_MA/internal/bootstrap"
)

func main() {
	application := bootstrap.NewApplication()
	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
