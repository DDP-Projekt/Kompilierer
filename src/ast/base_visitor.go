package ast

// when using the Visit* utility functions, this interface
// can be implemented to retreive the current module
type ModuleSetter interface {
	Visitor
	// called before a module is visited
	SetModule(*Module)
}

// when using the Visit* utility functions, this interface
// can be implemented to set the current scope
type ScopeSetter interface {
	Visitor
	// called before entering a scope in the AST
	SetScope(SymbolTable)
}

// when using the Visit* utility functions, this interface
// can be implemented to provide a condition to check if a node should be visited
type ConditionalVisitor interface {
	Visitor
	// Condition to check if a node should be visited
	ShouldVisit(Node) bool
}

// A base visitor that can be embedded in other visitors
// to provide a default implementation for all methods
// and other utility functionality
type BaseVisitor struct {
	// the current module being visited
	CurrentModule *Module
	// the current scope
	CurrentScope SymbolTable
	// Condition to check if a node should be visited
	// if nil, all nodes are visited
	VisitCondition func(Node) bool
}

var (
	_ Visitor            = (*BaseVisitor)(nil)
	_ ModuleSetter       = (*BaseVisitor)(nil)
	_ ScopeSetter        = (*BaseVisitor)(nil)
	_ ConditionalVisitor = (*BaseVisitor)(nil)
)

func (v *BaseVisitor) Visitor() {}

func (v *BaseVisitor) SetModule(m *Module) {
	v.CurrentModule = m
}

func (v *BaseVisitor) SetScope(s SymbolTable) {
	v.CurrentScope = s
}

func (v *BaseVisitor) ShouldVisit(n Node) bool {
	if v.VisitCondition == nil {
		return true
	}
	return v.VisitCondition(n)
}
