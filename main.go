package main

import (
	"um/app"
)

func main() {
	server := app.Routes{}
	server.StartGin()
}
