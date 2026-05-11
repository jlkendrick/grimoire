package descriptor

type PipelineDescriptor struct {
	CommandName string
	Steps       []StepDescriptor
	RitualHash  string
}

type StepDescriptor struct {
	Id        string
	SpellName string
	Params    map[string]any
}