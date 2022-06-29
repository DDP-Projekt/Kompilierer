package ast

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// simple visitor to print an AST
type printer struct {
	currentIdent int
	returned     string
}

// print the AST to stdout
func (ast *Ast) Print() {
	p := &printer{}
	WalkAst(ast, p)
	fmt.Println(p.returned)
}

func (ast *Ast) String() string {
	p := &printer{}
	WalkAst(ast, p)
	return p.returned
}

func (p *printer) printIdent() {
	for i := 0; i < p.currentIdent; i++ {
		p.print("   ")
	}
}

func (p *printer) print(s string) {
	p.returned += s
}

func (p *printer) parenthesizeNode(name string, nodes ...Node) string {
	p.print("(" + name)
	p.currentIdent++
	for _, node := range nodes {
		p.print("\n")
		p.printIdent()
		node.Accept(p)
	}
	p.currentIdent--
	if len(nodes) != 0 {
		p.printIdent()
	}
	p.print(")\n")
	return p.returned
}

func (p *printer) VisitBadDecl(d *BadDecl) Visitor {
	p.parenthesizeNode(fmt.Sprintf("BadDecl[%s]", d.Tok.String()))
	return p
}
func (p *printer) VisitVarDecl(d *VarDecl) Visitor {
	p.parenthesizeNode(fmt.Sprintf("VarDecl[%s]", d.Name.Literal), d.InitVal)
	return p
}
func (p *printer) VisitFuncDecl(d *FuncDecl) Visitor {
	p.parenthesizeNode(fmt.Sprintf("FuncDecl[%s: %v, %v, %s]", d.Name.Literal, tokenSlice(d.ParamNames).literals(), d.ParamTypes, d.Type.String()), d.Body)
	return p
}

func (p *printer) VisitBadExpr(e *BadExpr) Visitor {
	p.parenthesizeNode(fmt.Sprintf("BadExpr[%s]", e.Tok.String()))
	return p
}
func (p *printer) VisitIdent(e *Ident) Visitor {
	p.parenthesizeNode(fmt.Sprintf("Ident[%s]", e.Literal.Literal))
	return p
}
func (p *printer) VisitIntLit(e *IntLit) Visitor {
	p.parenthesizeNode(fmt.Sprintf("IntLit(%d)", e.Value))
	return p
}
func (p *printer) VisitFLoatLit(e *FloatLit) Visitor {
	p.parenthesizeNode(fmt.Sprintf("FloatLit(%f)", e.Value))
	return p
}
func (p *printer) VisitBoolLit(e *BoolLit) Visitor {
	p.parenthesizeNode(fmt.Sprintf("BoolLit(%v)", e.Value))
	return p
}
func (p *printer) VisitCharLit(e *CharLit) Visitor {
	p.parenthesizeNode(fmt.Sprintf("CharLit(%v)", e.Value))
	return p
}
func (p *printer) VisitStringLit(e *StringLit) Visitor {
	p.parenthesizeNode(fmt.Sprintf("StringLit[%s]", e.Token().Literal))
	return p
}
func (p *printer) VisitUnaryExpr(e *UnaryExpr) Visitor {
	p.parenthesizeNode(fmt.Sprintf("UnaryExpr[%s]", e.Operator.String()), e.Rhs)
	return p
}
func (p *printer) VisitBinaryExpr(e *BinaryExpr) Visitor {
	p.parenthesizeNode(fmt.Sprintf("BinaryExpr[%s]", e.Operator.String()), e.Lhs, e.Rhs)
	return p
}
func (p *printer) VisitTernaryExpr(e *TernaryExpr) Visitor {
	p.parenthesizeNode(fmt.Sprintf("TernaryExpr[%s]", e.Operator.String()), e.Lhs, e.Mid, e.Rhs)
	return p
}
func (p *printer) VisitGrouping(e *Grouping) Visitor {
	p.parenthesizeNode("Grouping", e.Expr)
	return p
}
func (p *printer) VisitFuncCall(e *FuncCall) Visitor {
	args := make([]Node, 0)
	for _, v := range e.Args {
		args = append(args, v)
	}
	p.parenthesizeNode(fmt.Sprintf("FuncCall(%s)", e.Name), args...)
	return p
}

func (p *printer) VisitBadStmt(s *BadStmt) Visitor {
	p.parenthesizeNode(fmt.Sprintf("BadStmt[%s]", s.Tok.String()))
	return p
}
func (p *printer) VisitDeclStmt(s *DeclStmt) Visitor {
	p.parenthesizeNode("DeclStmt", s.Decl)
	return p
}
func (p *printer) VisitExprStmt(s *ExprStmt) Visitor {
	p.parenthesizeNode("ExprStmt", s.Expr)
	return p
}
func (p *printer) VisitAssignStmt(s *AssignStmt) Visitor {
	p.parenthesizeNode(fmt.Sprintf("AssignStmt[%s]", s.Name.Literal), s.Rhs)
	return p
}
func (p *printer) VisitBlockStmt(s *BlockStmt) Visitor {
	args := make([]Node, len(s.Statements))
	for i, v := range s.Statements {
		args[i] = v
	}
	p.parenthesizeNode("BlockStmt", args...)
	return p
}
func (p *printer) VisitIfStmt(s *IfStmt) Visitor {
	if s.Else != nil {
		p.parenthesizeNode("IfStmt", s.Condition, s.Then, s.Else)
	} else {
		p.parenthesizeNode("IfStmt", s.Condition, s.Then)
	}
	return p
}
func (p *printer) VisitWhileStmt(s *WhileStmt) Visitor {
	p.parenthesizeNode("WhileStmt", s.Condition, s.Body)
	return p
}
func (p *printer) VisitForStmt(s *ForStmt) Visitor {
	p.parenthesizeNode("ForStmt", s.Initializer, s.To, s.StepSize, s.Body)
	return p
}
func (p *printer) VisitForRangeStmt(s *ForRangeStmt) Visitor {
	p.parenthesizeNode("ForRangeStmt", s.Initializer, s.In, s.Body)
	return p
}
func (p *printer) VisitFuncCallStmt(s *FuncCallStmt) Visitor {
	p.parenthesizeNode("FuncCallStmt", s.Call)
	return p
}
func (p *printer) VisitReturnStmt(s *ReturnStmt) Visitor {
	p.parenthesizeNode("ReturnStmt", s.Value)
	return p
}

type tokenSlice []token.Token

func (t tokenSlice) literals() []string {
	result := make([]string, 0, len(t))
	for _, v := range t {
		result = append(result, v.Literal)
	}
	return result
}
