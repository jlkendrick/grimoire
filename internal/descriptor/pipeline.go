package descriptor

type PipelineDescriptor struct {
	CommandName string
	Steps       []StepDescriptor
	RitualHash  string
}

// StepDescriptor is either a spell-step (SpellName set) or an if-step
// (Condition set). The reconciler enforces the discriminator; nothing
// downstream should see a malformed mix.
type StepDescriptor struct {
	Id        string         `json:",omitempty"`
	SpellName string         `json:",omitempty"`
	Params    map[string]any `json:",omitempty"`

	Condition string           `json:",omitempty"`
	Then      []StepDescriptor `json:",omitempty"`
	Else      []StepDescriptor `json:",omitempty"`
}

func (s StepDescriptor) Kind() string {
	if s.Condition != "" {
		return "if"
	}
	return "spell"
}
