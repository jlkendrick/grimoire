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

func TestPipe2(input1 string, input2 string) (string, string) {
	// fmt.Printf("TestPipe2: input1 = %s, input2 = %s\n", input1, input2)
	return input1 + input2, input2 + input1
}

func TestPipe3(input_a string, input_b string) string {
	return input_a + input_b
}