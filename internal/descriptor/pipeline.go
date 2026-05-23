package descriptor

type PipelineDescriptor struct {
	CommandName string
	Steps       []StepDescriptor
	RitualHash  string
}

// StepDescriptor is one of three kinds: spell-step (SpellName set),
// if-step (Condition set), or let-step (Let set, with Value carrying
// the expression source). The reconciler enforces the discriminator;
// nothing downstream should see a malformed mix.
type StepDescriptor struct {
	Id        string         `json:",omitempty"`
	SpellName string         `json:",omitempty"`
	Params    map[string]any `json:",omitempty"`

	Condition string           `json:",omitempty"`
	Then      []StepDescriptor `json:",omitempty"`
	Else      []StepDescriptor `json:",omitempty"`

	Let   string `json:",omitempty"`
	Value string `json:",omitempty"`
}

func (s StepDescriptor) Kind() string {
	if s.Let != "" {
		return "let"
	}
	if s.Condition != "" {
		return "if"
	}
	return "spell"
}
