package descriptor

type PipelineDescriptor struct {
	CommandName string
	Steps       []StepDescriptor
}

type StepDescriptor struct {
	SpellName string
	Params    []ParamDescriptor
}