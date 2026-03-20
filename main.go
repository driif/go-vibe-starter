package main

import (
	"github.com/driif/go-vibe-starter/cmd"
	// Import postgres driver for database/sql package
	_ "github.com/lib/pq"
)

func main() {
	cmd.Execute()
}
