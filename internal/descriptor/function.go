package descriptor

type FunctionDescriptor struct {
	CommandName         string
	FunctionName        string
	AbsPathToSourceFile string
	RelPathToSourceFile string
	ScrollPath          string
	Interpreter         string
	SourceHash          string
	SpellHash           string
	Params              []ParamDescriptor
	// Returns is the function's declared return type; nil means
	// unannotated (Unknown), which downstream validation treats as
	// compatible with everything — annotations only ever add checking.
	Returns *TypeInfo `json:",omitempty"`
}

type ParamDescriptor struct {
	Name           string
	RawTypeText    string     // exactly what appeared in source: "List[User]" or ""
	ResolvedType   *TypeInfo  // nil if extractor couldn't resolve
	Default        any  			// nil if no default
	ExtractorNotes []string   // "type inferred from default", "untyped parameter", etc.
}

type TypeInfo struct {
	Kind     TypeKind   // Primitive | List | Map | Struct | Optional | Unknown
	Element  *TypeInfo  // for List, Optional
	Name     string     // "int", "str", "DeployConfig"
	// Struct fields coming soon
}

type TypeKind string
const (
	TypeKindPrimitive TypeKind = "primitive"
	TypeKindList TypeKind = "list"
	TypeKindMap TypeKind = "map"
	TypeKindStruct TypeKind = "struct"
	TypeKindOptional TypeKind = "optional"
	TypeKindNone TypeKind = "none"
	TypeKindUnknown TypeKind = "unknown"
)