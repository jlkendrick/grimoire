package main

import (
	"fmt"
	"time"

	cmd "github.com/jlkendrick/grimoire/cmd"
)

func main() {
	start_time := time.Now()
	cmd.Execute()
	end_time := time.Now()
	duration := end_time.Sub(start_time)
	fmt.Printf("Command executed in %v\n", duration)
}