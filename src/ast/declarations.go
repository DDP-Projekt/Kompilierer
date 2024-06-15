package ast

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type (
	// an invalid Declaration
	BadDecl struct {
		Tok token.Token
		Err ddperror.Error
		Mod *Module
	}

	VarDecl struct {
		Range           token.Range
		CommentTok      *token.Token  // optional comment (also contained in ast.Comments)
		Type            ddptypes.Type // type of the variable
		NameTok         token.Token   // identifier name
		IsPublic        bool          // wether the function is marked with öffentliche
		IsExternVisible bool          // wether the variable is marked as extern visible
		Mod             *Module       // the module in which the variable was declared
		InitVal         Expression    // initial value
	}

	FuncDecl struct {
		Range           token.Range
		CommentTok      *token.Token    // optional comment (also contained in ast.Comments)
		Tok             token.Token     // Die
		NameTok         token.Token     // token of the name
		IsPublic        bool            // wether the function is marked with öffentliche
		IsExternVisible bool            // wether the function is marked as extern visible
		Mod             *Module         // the module in which the function was declared
		Parameters      []ParameterInfo // name, type and comments of parameters
		Type            ddptypes.Type   // return Type, Zahl Kommazahl nichts ...
		Body            *BlockStmt      // nil for extern functions
		ExternFile      token.Token     // string literal with filepath (only pesent if Body is nil)
		Aliases         []*FuncAlias
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
)

func (decl *BadDecl) node()    {}
func (decl *VarDecl) node()    {}
func (decl *FuncDecl) node()   {}
func (decl *StructDecl) node() {}

func (decl *BadDecl) String() string    { return "BadDecl" }
func (decl *VarDecl) String() string    { return "VarDecl" }
func (decl *FuncDecl) String() string   { return "FuncDecl" }
func (decl *StructDecl) String() string { return "StructDecl" }

func (decl *BadDecl) Token() token.Token    { return decl.Tok }
func (decl *VarDecl) Token() token.Token    { return decl.NameTok }
func (decl *FuncDecl) Token() token.Token   { return decl.Tok }
func (decl *StructDecl) Token() token.Token { return decl.Tok }

func (decl *BadDecl) GetRange() token.Range    { return decl.Err.Range }
func (decl *VarDecl) GetRange() token.Range    { return decl.Range }
func (decl *FuncDecl) GetRange() token.Range   { return decl.Range }
func (decl *StructDecl) GetRange() token.Range { return decl.Range }

func (decl *BadDecl) Accept(visitor FullVisitor) VisitResult    { return visitor.VisitBadDecl(decl) }
func (decl *VarDecl) Accept(visitor FullVisitor) VisitResult    { return visitor.VisitVarDecl(decl) }
func (decl *FuncDecl) Accept(visitor FullVisitor) VisitResult   { return visitor.VisitFuncDecl(decl) }
func (decl *StructDecl) Accept(visitor FullVisitor) VisitResult { return visitor.VisitStructDecl(decl) }

func (decl *BadDecl) declarationNode()    {}
func (decl *VarDecl) declarationNode()    {}
func (decl *FuncDecl) declarationNode()   {}
func (decl *StructDecl) declarationNode() {}

func (decl *BadDecl) Name() string    { return "" }
func (decl *VarDecl) Name() string    { return decl.NameTok.Literal }
func (decl *FuncDecl) Name() string   { return decl.NameTok.Literal }
func (decl *StructDecl) Name() string { return decl.NameTok.Literal }

func (decl *BadDecl) Public() bool    { return false }
func (decl *VarDecl) Public() bool    { return decl.IsPublic }
func (decl *FuncDecl) Public() bool   { return decl.IsPublic }
func (decl *StructDecl) Public() bool { return decl.IsPublic }

func (decl *BadDecl) Comment() *token.Token    { return nil }
func (decl *VarDecl) Comment() *token.Token    { return decl.CommentTok }
func (decl *FuncDecl) Comment() *token.Token   { return decl.CommentTok }
func (decl *StructDecl) Comment() *token.Token { return decl.CommentTok }

func (decl *BadDecl) Module() *Module    { return decl.Mod }
func (decl *VarDecl) Module() *Module    { return decl.Mod }
func (decl *FuncDecl) Module() *Module   { return decl.Mod }
func (decl *StructDecl) Module() *Module { return decl.Mod }
