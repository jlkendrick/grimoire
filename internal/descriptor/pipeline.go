package descriptor

type PipelineDescriptor struct {
	CommandName string
	Steps       []StepDescriptor
	RitualHash  string
}

type StepDescriptor struct {
	SpellName string
	Params    []ParamDescriptor
}