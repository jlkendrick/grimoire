package descriptor

type FunctionDescriptor struct {
	CommandName   string
	FunctionName  string
	SourceFile    string
	ScrollPath    string
	Params        []ParamDescriptor
	Interpreter   string
	// Return type coming soon
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
	TypeKindUnknown TypeKind = "unknown"
)