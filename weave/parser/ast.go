package parser

// 1. The Root Node
type Ritual struct {
	Name string  `parser:"'ritual' @Ident '{'"`
	Body []*Stmt `parser:"@@* '}'"`
}

// 2. Statements
type Stmt struct {
	Let  *LetStmt  `parser:"@@"`
	If   *IfStmt   `parser:"| @@"`
	Call *CallExpr `parser:"| @@"`
}

// 3. Variable Assignment
type LetStmt struct {
	Ident string `parser:"'let' @Ident '='"`
	Expr  *Expr  `parser:"@@"`
}

// 4. Control Flow
type IfStmt struct {
	Condition *Expr   `parser:"'if' @@ '{'"`
	TrueBlock []*Stmt `parser:"@@* '}'"`
	ElseBlock []*Stmt `parser:"( 'else' '{' @@* '}' )?"` // Optional else block
}

// 5. Function Calls
//
// An optional `module.` prefix lets a ritual call into a spell defined in
// another registered scroll: bare `deploy()` resolves against the ritual's
// own scroll; qualified `B.deploy()` resolves against scroll B's cache.
type CallExpr struct {
	Module string      `parser:"( @Ident '.' )?"`
	Name   string      `parser:"@Ident '('"`
	Args   []*Argument `parser:"( @@ ( ',' @@ )* )? ')'"` // Comma-separated arguments
}

type Argument struct {
	Name  string `parser:"( @Ident '=' )?"` // Optional named parameter (e.g., celsius =)
	Value *Expr  `parser:"@@"`
}

// 6. Expressions (Logical and Equality operators)
type Expr struct {
	Left  *Term      `parser:"@@"`
	Right []*OpRight `parser:"@@*"`
}

type OpRight struct {
	Operator string `parser:"@( '|' '|' | '=' '=' )"` // Captures || or ==
	Right    *Term  `parser:"@@"`
}

// 7. Base Values
//
// Boolean precedes Ref so that the literals `true` and `false` are captured
// as bools rather than as Ident-rooted references. Participle tries
// alternatives in declaration order, and `@Ident` would otherwise consume
// `true`/`false` before the boolean branch ever runs.
type Term struct {
	Call    *CallExpr  `parser:"@@"`
	Boolean *bool      `parser:"| ( @'true' | @'false' )"`
	List    *ListLit   `parser:"| @@"`   // [a, b, c]
	Object  *ObjectLit `parser:"| @@"`   // { key = value, ... }
	Ref     *RefExpr   `parser:"| @@"`   // e.g., temp.freezing
	String  *string    `parser:"| @String"`
	Float   *float64   `parser:"| @Float"`
	Int     *int       `parser:"| @Int"`
}

// ListLit and ObjectLit are collection literals. They are valid only as a
// direct call-argument value (e.g. `f(tags = [1, 2])`), not inside operator
// expressions or if/let conditions — the transpiler enforces this since the
// scroll condition grammar has no notion of collection literals.
type ListLit struct {
	Elements []*Expr `parser:"'[' ( @@ ( ',' @@ )* )? ']'"`
}

type ObjectLit struct {
	Entries []*ObjectEntry `parser:"'{' ( @@ ( ',' @@ )* )? '}'"`
}

type ObjectEntry struct {
	Key   string `parser:"( @Ident | @String ) '='"`
	Value *Expr  `parser:"@@"`
}

// Handles dot-notation and array indexing
type RefExpr struct {
	Root  string      `parser:"@Ident"`
	Chain []*AccessOp `parser:"@@*"`
}

// AccessOp represents a single step down the data structure
type AccessOp struct {
	Property *string `parser:"'.' @Ident"`   // e.g., .id
	Index    *int    `parser:"| '[' @Int ']'"` // e.g., [0]
}
