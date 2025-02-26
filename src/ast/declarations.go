package ast

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type AliasTrie = *at.Trie[*token.Token, Alias]

type (
	// an invalid Declaration
	BadDecl struct {
		Tok token.Token
		Err ddperror.Error
		Mod *Module
	}

	ConstDecl struct {
		Range      token.Range
		Mod        *Module       // the module in which the variable was declared
		CommentTok *token.Token  // optional comment (also contained in ast.Comments)
		Type       ddptypes.Type // type of the variable
		NameTok    token.Token   // identifier name
		Val        Literal
		IsPublic   bool // wether the function is marked with öffentliche
	}

	VarDecl struct {
		Range           token.Range
		CommentTok      *token.Token  // optional comment (also contained in ast.Comments)
		Type            ddptypes.Type // type of the variable
		NameTok         token.Token   // identifier name
		TypeRange       token.Range   // range of the type (mainly used by the LSP)
		IsPublic        bool          // wether the function is marked with öffentliche
		IsExternVisible bool          // wether the variable is marked as extern visible
		Mod             *Module       // the module in which the variable was declared
		InitVal         Expression    // initial value
		InitType        ddptypes.Type // type of InitVal, filled in by the typechecker, used to keep information about typedefs
	}

	FuncDecl struct {
		Range           token.Range
		CommentTok      *token.Token     // optional comment (also contained in ast.Comments)
		Tok             token.Token      // Die
		NameTok         token.Token      // token of the name
		IsPublic        bool             // wether the function is marked with öffentliche
		IsExternVisible bool             // wether the function is marked as extern visible
		Mod             *Module          // the module in which the function was declared
		Parameters      []ParameterInfo  // name, type and comments of parameters
		ReturnType      ddptypes.Type    // return Type, Zahl Kommazahl nichts ...
		ReturnTypeRange token.Range      // range of the return type (mainly used by the LSP)
		Body            *BlockStmt       // nil for extern functions or forward declarations
		Def             *FuncDef         // non-nil for forward declarations
		ExternFile      token.Token      // string literal with filepath (only pesent if Body is nil)
		Operator        Operator         // the operator this function overloads, or nil if it does not overload an operator
		Generic         *GenericFuncInfo // only filled if the function was declared as generic
		Aliases         []*FuncAlias
	}

	// holds information about a generic function declaration
	GenericFuncInfo struct {
		Types          map[string]*ddptypes.GenericType // all declared generic types
		Tokens         []token.Token                    // tokens of the body that need to be parsed
		Context        GenericContext                   // context up to the point where the function was declared
		Instantiations map[*Module][]*FuncDecl          // all existing instantiations for a given module
	}

	// the context captured by a generic function declaration
	// to parse an instantiation it has to be merged with the symbols and aliases at
	// the point of instantiation
	GenericContext struct {
		Symbols   SymbolTable
		Aliases   AliasTrie
		Operators map[Operator][]*FuncDecl
	}

	// is a statement and not a declaration but grouped in this File with FuncDecl for readability
	FuncDef struct {
		Range token.Range
		Tok   token.Token // Die
		Func  *FuncDecl
		Body  *BlockStmt
	}

	StructDecl struct {
		Range      token.Range
		CommentTok *token.Token // optional comment (also contained in ast.Comments)
		Tok        token.Token  // Wir
		NameTok    token.Token  // token of the name
		IsPublic   bool         // wether the struct decl is marked with öffentliche
		Mod        *Module      // the module in which the struct was declared
		// Field declarations of the struct in order of declaration
		// only contains *VarDecl and *BadDecl s
		Fields  []Declaration
		Type    *ddptypes.StructType // the type resulting from this decl
		Aliases []*StructAlias       // the constructors of the struct
	}

	TypeAliasDecl struct {
		Range           token.Range
		CommentTok      *token.Token // optional comment
		Tok             token.Token  // Wir
		NameTok         token.Token  // token of the name
		IsPublic        bool
		Mod             *Module
		Underlying      ddptypes.Type       // the underlying type
		UnderlyingRange token.Range         // range of the underlying type (mainly used by the LSP)
		Type            *ddptypes.TypeAlias // the resulting  TypeAlias type
	}

	TypeDefDecl struct {
		Range           token.Range
		CommentTok      *token.Token // optional comment
		Tok             token.Token  // Wir
		NameTok         token.Token  // token of the name
		IsPublic        bool
		Mod             *Module
		Underlying      ddptypes.Type     // the underlying type
		UnderlyingRange token.Range       // range of the underlying type (mainly used by the LSP)
		Type            *ddptypes.TypeDef // the resulting  TypeDef type
	}
)

func (decl *BadDecl) node()       {}
func (decl *ConstDecl) node()     {}
func (decl *VarDecl) node()       {}
func (decl *FuncDecl) node()      {}
func (decl *FuncDef) node()       {}
func (decl *StructDecl) node()    {}
func (decl *TypeAliasDecl) node() {}
func (decl *TypeDefDecl) node()   {}

func (decl *BadDecl) String() string       { return "BadDecl" }
func (decl *ConstDecl) String() string     { return "ConstDecl" }
func (decl *VarDecl) String() string       { return "VarDecl" }
func (decl *FuncDecl) String() string      { return "FuncDecl" }
func (decl *FuncDef) String() string       { return "FuncDef" }
func (decl *StructDecl) String() string    { return "StructDecl" }
func (decl *TypeAliasDecl) String() string { return "TypeAliasDecl" }
func (decl *TypeDefDecl) String() string   { return "TypeDefDecl" }

func (decl *BadDecl) Token() token.Token       { return decl.Tok }
func (decl *ConstDecl) Token() token.Token     { return decl.NameTok }
func (decl *VarDecl) Token() token.Token       { return decl.NameTok }
func (decl *FuncDecl) Token() token.Token      { return decl.Tok }
func (decl *FuncDef) Token() token.Token       { return decl.Tok }
func (decl *StructDecl) Token() token.Token    { return decl.Tok }
func (decl *TypeAliasDecl) Token() token.Token { return decl.Tok }
func (decl *TypeDefDecl) Token() token.Token   { return decl.Tok }

func (decl *BadDecl) GetRange() token.Range       { return decl.Err.Range }
func (decl *ConstDecl) GetRange() token.Range     { return decl.Range }
func (decl *VarDecl) GetRange() token.Range       { return decl.Range }
func (decl *FuncDecl) GetRange() token.Range      { return decl.Range }
func (decl *FuncDef) GetRange() token.Range       { return decl.Range }
func (decl *StructDecl) GetRange() token.Range    { return decl.Range }
func (decl *TypeAliasDecl) GetRange() token.Range { return decl.Range }
func (decl *TypeDefDecl) GetRange() token.Range   { return decl.Range }

func (decl *BadDecl) Accept(visitor FullVisitor) VisitResult    { return visitor.VisitBadDecl(decl) }
func (decl *ConstDecl) Accept(visitor FullVisitor) VisitResult  { return visitor.VisitConstDecl(decl) }
func (decl *VarDecl) Accept(visitor FullVisitor) VisitResult    { return visitor.VisitVarDecl(decl) }
func (decl *FuncDecl) Accept(visitor FullVisitor) VisitResult   { return visitor.VisitFuncDecl(decl) }
func (decl *FuncDef) Accept(visitor FullVisitor) VisitResult    { return visitor.VisitFuncDef(decl) }
func (decl *StructDecl) Accept(visitor FullVisitor) VisitResult { return visitor.VisitStructDecl(decl) }
func (decl *TypeAliasDecl) Accept(visitor FullVisitor) VisitResult {
	return visitor.VisitTypeAliasDecl(decl)
}

func (decl *TypeDefDecl) Accept(visitor FullVisitor) VisitResult {
	return visitor.VisitTypeDefDecl(decl)
}

func (decl *BadDecl) declarationNode()       {}
func (decl *ConstDecl) declarationNode()     {}
func (decl *VarDecl) declarationNode()       {}
func (decl *FuncDecl) declarationNode()      {}
func (decl *FuncDef) statementNode()         {}
func (decl *StructDecl) declarationNode()    {}
func (decl *TypeAliasDecl) declarationNode() {}
func (decl *TypeDefDecl) declarationNode()   {}

func (decl *BadDecl) Name() string       { return "" }
func (decl *ConstDecl) Name() string     { return decl.NameTok.Literal }
func (decl *VarDecl) Name() string       { return decl.NameTok.Literal }
func (decl *FuncDecl) Name() string      { return decl.NameTok.Literal }
func (decl *StructDecl) Name() string    { return decl.NameTok.Literal }
func (decl *TypeAliasDecl) Name() string { return decl.NameTok.Literal }
func (decl *TypeDefDecl) Name() string   { return decl.NameTok.Literal }

func (decl *BadDecl) Public() bool       { return false }
func (decl *ConstDecl) Public() bool     { return decl.IsPublic }
func (decl *VarDecl) Public() bool       { return decl.IsPublic }
func (decl *FuncDecl) Public() bool      { return decl.IsPublic }
func (decl *StructDecl) Public() bool    { return decl.IsPublic }
func (decl *TypeAliasDecl) Public() bool { return decl.IsPublic }
func (decl *TypeDefDecl) Public() bool   { return decl.IsPublic }

func (decl *BadDecl) Comment() *token.Token       { return nil }
func (decl *ConstDecl) Comment() *token.Token     { return decl.CommentTok }
func (decl *VarDecl) Comment() *token.Token       { return decl.CommentTok }
func (decl *FuncDecl) Comment() *token.Token      { return decl.CommentTok }
func (decl *StructDecl) Comment() *token.Token    { return decl.CommentTok }
func (decl *TypeAliasDecl) Comment() *token.Token { return decl.CommentTok }
func (decl *TypeDefDecl) Comment() *token.Token   { return decl.CommentTok }

func (decl *BadDecl) Module() *Module       { return decl.Mod }
func (decl *ConstDecl) Module() *Module     { return decl.Mod }
func (decl *VarDecl) Module() *Module       { return decl.Mod }
func (decl *FuncDecl) Module() *Module      { return decl.Mod }
func (decl *StructDecl) Module() *Module    { return decl.Mod }
func (decl *TypeAliasDecl) Module() *Module { return decl.Mod }
func (decl *TypeDefDecl) Module() *Module   { return decl.Mod }
