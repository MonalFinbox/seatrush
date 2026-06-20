package main

import (
	"fmt"

	"github.com/MonalFinbox/seatrush/internal/config"
)

func main() {
	cfg := config.Load()
	fmt.Printf("SeatRush starting on port %s\n", cfg.ServerPort)
}
