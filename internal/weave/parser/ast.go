package parser

// 1. The Root Node
type Ritual struct {
	Name string  `"ritual" @Ident "{"`
	Body []*Stmt `@@* "}"`
}

// 2. Statements
type Stmt struct {
	Let  *LetStmt  `  @@`
	If   *IfStmt   `| @@`
	Call *CallExpr `| @@`
}

// 3. Variable Assignment
type LetStmt struct {
	Ident string `"let" @Ident "="`
	Expr  *Expr  `@@`
}

// 4. Control Flow
type IfStmt struct {
	Condition *Expr   `"if" @@ "{"`
	TrueBlock []*Stmt `@@* "}"`
	ElseBlock []*Stmt `( "else" "{" @@* "}" )?` // Optional else block
}

// 5. Function Calls
type CallExpr struct {
	Name string      `@Ident "("`
	Args []*Argument `( @@ ( "," @@ )* )? ")"` // Comma-separated arguments
}

type Argument struct {
	Name  string `( @Ident "=" )?` // Optional named parameter (e.g., celsius =)
	Value *Expr  `@@`
}

// 6. Expressions (Logical and Equality operators)
type Expr struct {
	Left  *Term      `@@`
	Right []*OpRight `@@*`
}

type OpRight struct {
	Operator string `@( "|" "|" | "=" "=" )` // Captures || or ==
	Right    *Term  `@@`
}

// 7. Base Values
type Term struct {
	Call    *CallExpr `  @@`
	Ref     *RefExpr  `| @@` // e.g., temp.freezing
	String  *string   `| @String`
	Int     *int      `| @Int`
	Boolean *bool     `| ( @"true" | @"false" )`
}

// Handles dot-notation and array indexing
type RefExpr struct {
	Root  string      `@Ident`
	Chain []*AccessOp `@@*`
}

// AccessOp represents a single step down the data structure
type AccessOp struct {
	Property *string `  "." @Ident`    // e.g., .id
	Index    *int    `| "[" @Int "]"`  // e.g., [0]
}