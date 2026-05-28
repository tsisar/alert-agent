package main

import (
	"github.com/joho/godotenv"
	"github.com/tsisar/alert-agent/cmd/alert-agent/cmd"
)

func main() {
	_ = godotenv.Load()
	cmd.Execute()
}
