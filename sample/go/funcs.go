package sample

import (
	"fmt"
	"hash/fnv"
	"os"
	"time"
)

func HelloWorld() (string, string) {
	for i := range 10 {
		fmt.Printf("stdout tick %d/%d\n", i+1, 10)
		fmt.Fprintf(os.Stderr, "stderr tick %d/%d\n", i+1, 10)
		time.Sleep(500 * time.Millisecond)
	}

	return "Hello", "World!"
}

func TestPipe() (string, string) {
	return "Output11", "Output12"
}

func TestPipe2(input1 string, input2 string) (string, string) {
	fmt.Fprintf(os.Stderr, "TestPipe2: input1 = %s, input2 = %s\n", input1, input2)
	return input1 + input2, input2 + input1
}

func TestPipe3(input_a string, input_b string) string {
	return input_a + input_b
}

var heroes = []string{
	"a tired detective",
	"the apprentice cartographer",
	"a sleepless physician",
	"the mayor's lost daughter",
	"a freelance translator",
}

var villains = []string{
	"the smiling notary",
	"the regional vice-consul",
	"a man who calls himself Theo",
	"the woman in the brass mask",
	"your own shadow",
}

var mcguffins = []string{
	"a brass key",
	"a sealed envelope",
	"the wrong address",
	"a single photograph",
	"a list of seven names",
}

func pickFrom(items []string, key string) string {
	h := fnv.New64a()
	h.Write([]byte(key))
	return items[h.Sum64()%uint64(len(items))]
}

func ForgeCharacters(setting string, mood string) (string, string, string) {
	base := setting + "|" + mood
	return pickFrom(heroes, base+"|hero"),
		pickFrom(villains, base+"|villain"),
		pickFrom(mcguffins, base+"|mcguffin")
}