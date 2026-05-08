package sample

import (
	"os"
	"fmt"
)

func HelloWorld(n int) string {
	for i := range n {
		fmt.Printf("stdout tick %d/%d\n", i+1, n)
		fmt.Fprintf(os.Stderr, "stderr tick %d/%d\n", i+1, n)
		// time.Sleep(500 * time.Millisecond)
	}

	return "Hello, World!"
}

func TestPipe() (string, string) {
	return "Output11", "Output12"
}