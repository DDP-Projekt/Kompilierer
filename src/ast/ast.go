package ast

import (
	"fmt"
	"runtime/debug"

	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// represents an Abstract Syntax Tree for a DDP program
type Ast struct {
	Statements []Statement   // the top level statements
	Comments   []token.Token // all the comments in the source code
	Symbols    SymbolTable
	Faulty     bool              // set if the ast has any errors (doesn't matter what from which phase they came)
	metadata   map[Node]Metadata // metadata for each node
}

// returns all the metadata attached to the given node
func (ast *Ast) GetMetadata(node Node) (Metadata, bool) {
	md, ok := ast.metadata[node]
	return md, ok
}

// returns the metadata of the given kind attached to the given node
func (ast *Ast) GetMetadataByKind(node Node, kind MetadataKind) (MetadataAttachment, bool) {
	md, ok := ast.GetMetadata(node)
	return md.Attachments[kind], ok
}

// adds metadata to the given node
func (ast *Ast) AddAttachement(node Node, attachment MetadataAttachment) {
	if ast.metadata == nil {
		ast.metadata = make(map[Node]Metadata, 8)
	}

	md := ast.metadata[node]
	if md.Attachments == nil {
		md.Attachments = make(map[MetadataKind]MetadataAttachment)
	}
	md.Attachments[attachment.Kind()] = attachment
	ast.metadata[node] = md
}

// removes metadata of the given kind from the given node
func (ast *Ast) RemoveAttachment(node Node, kind MetadataKind) {
	md, ok := ast.GetMetadata(node)
	if ok {
		delete(md.Attachments, kind)
		ast.metadata[node] = md
	}
}

// returns a string representation of the AST as S-Expressions
func (ast *Ast) String() string {
	defer panic_wrapper()
	printer := &printer{ast: ast}
	for _, stmt := range ast.Statements {
		stmt.Accept(printer)
	}
	return printer.returned
}

// print the AST to stdout
func (ast *Ast) Print() {
	fmt.Println(ast.String())
}

// wraps a panic with more information and re-panics
func panic_wrapper() {
	if err := recover(); err != nil {
		stack_trace := debug.Stack()
		wraps, _ := err.(error)
		panic(fmt.Errorf("%w\nStack Trace: %s", wraps, string(stack_trace)))
	}
}

type (
	// interface for a alias
	// of either a function
	// or a struct constructor
	Alias interface {
		// tokens of the alias
		GetTokens() []token.Token
		// tokens of the alias but as slice of *token.Token and without the EOF
		// this is the slice that is used as keys in the AliasTrie
		GetKey() []*token.Token
		// the original string
		GetOriginal() token.Token
		// *FuncDecl or *StructDecl
		Decl() Declaration
		// types of the arguments (used for funcCall parsing)
		GetArgs() map[string]ddptypes.ParameterType
	}

	// wrapper for a function alias
	FuncAlias struct {
		Tokens   []token.Token                     // tokens of the alias
		pTokens  []*token.Token                    // cache for GetPTokens
		Original token.Token                       // the original string
		Func     *FuncDecl                         // the function it refers to (if it is used outside a FuncDecl)
		Args     map[string]ddptypes.ParameterType // types of the arguments (used for funcCall parsing)
		Negated  bool                              // if the alias has been negated
	}

	// wrapper for a struct alias
	StructAlias struct {
		Tokens   []token.Token            // tokens of the alias
		pTokens  []*token.Token           // cache for GetPTokens
		Original token.Token              // the original string
		Struct   *StructDecl              // the struct decl it refers to
		Args     map[string]ddptypes.Type // types of the arguments (only those that the alias needs)
	}
)

func (alias *FuncAlias) GetTokens() []token.Token {
	return alias.Tokens
}

func (alias *FuncAlias) GetKey() []*token.Token {
	if alias.pTokens == nil {
		alias.pTokens = toPointerSlice(alias.Tokens[:len(alias.Tokens)-1])
	}
	return alias.pTokens
}

func (alias *FuncAlias) GetOriginal() token.Token {
	return alias.Original
}

func (alias *FuncAlias) Decl() Declaration {
	return alias.Func
}

func (alias *FuncAlias) GetArgs() map[string]ddptypes.ParameterType {
	return alias.Args
}

func (alias *StructAlias) GetTokens() []token.Token {
	return alias.Tokens
}

func (alias *StructAlias) GetKey() []*token.Token {
	if alias.pTokens == nil {
		alias.pTokens = toPointerSlice(alias.Tokens[:len(alias.Tokens)-1])
	}
	return alias.pTokens
}

func (alias *StructAlias) GetOriginal() token.Token {
	return alias.Original
}

func (alias *StructAlias) Decl() Declaration {
	return alias.Struct
}

func (alias *StructAlias) GetArgs() map[string]ddptypes.ParameterType {
	paramTypes := make(map[string]ddptypes.ParameterType, len(alias.Args))
	for name, arg := range alias.Args {
		paramTypes[name] = ddptypes.ParameterType{
			Type:        arg,
			IsReference: false,
		}
	}
	return paramTypes
}

// holds all information about a single function parameter
type ParameterInfo struct {
	Name      token.Token            // the name token of the parameter
	Type      ddptypes.ParameterType // the type of the parameter or default value if there was an error during parsing
	TypeRange token.Range            // range of the type (mainly for the LSP)
	Comment   *token.Token           // the comment token, or nil if none was present
}

// wether the ParameterInfo's type is not the default value (i.e. was not parsed)
func (param *ParameterInfo) HasValidType() bool {
	return param.Type != ddptypes.ParameterType{}
}

//go-sumtype:decl Node
//go-sumtype:decl Expression
//go-sumtype:decl Statement
//go-sumtype:decl Declaration
//go-sumtype:decl Assigneable

// basic Node interfaces
type (
	Node interface {
		fmt.Stringer
		node() // dummy function for the interface
		Token() token.Token
		GetRange() token.Range
		Accept(FullVisitor) VisitResult
	}

	Expression interface {
		Node
		expressionNode() // dummy function for the interface
	}

	Statement interface {
		Node
		statementNode() // dummy function for the interface
	}

	Declaration interface {
		Node
		declarationNode()      // dummy function for the interface
		Name() string          // returns the name of the declaration or "" for BadDecls
		Public() bool          // returns wether the declaration is public. always false for BadDecls
		Comment() *token.Token // returns a optional comment
		Module() *Module       // returns the module from which the declaration comes
	}

	// *Ident or *Indexing
	// Nodes that fulfill this interface can be
	// on the left side of an assignement (meaning, variables or references)
	Assigneable interface {
		Expression
		assigneable() // dummy function for the interface
	}
)
