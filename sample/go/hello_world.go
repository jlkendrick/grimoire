package sample

import (
	"fmt"
	"os"
)

func HelloWorld(n int) string {
	for i := range n {
		fmt.Printf("stdout tick %d/%d\n", i+1, n)
		fmt.Fprintf(os.Stderr, "stderr tick %d/%d\n", i+1, n)
		// time.Sleep(1 * time.Second)
	}

	return "Hello, World!"
}