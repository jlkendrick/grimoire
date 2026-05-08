package scroll

type Ritual struct {
	Command string `yaml:"command"`
	Steps   []Step `yaml:"steps"`
}

type Step struct {
	Spell string `yaml:"spell"`
}