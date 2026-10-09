package refops

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
)

// NodeRefOp describes what an expression node itself does with references,
// before any implicit casts of the node are applied.
type NodeRefOp int

const (
	// the node yields its value
	OpValue NodeRefOp = iota
	// the node yields the address of the storage it designates (an lvalue),
	// e.g. an assignable Ident, field access, list index, assignable 'Wert von' or assignable cast
	OpAddr
	// the node dereferences its operand once ('Wert von', cast from a reference to a value)
	OpDeref
	// the node copies its operand into a new reference cell (cast from a value to a reference)
	OpPromote
)

func (op NodeRefOp) String() string {
	switch op {
	case OpValue:
		return "Value"
	case OpAddr:
		return "Addr"
	case OpDeref:
		return "Deref"
	case OpPromote:
		return "Promote"
	}
	return fmt.Sprintf("NodeRefOp(%d)", int(op))
}

// RefOpsMeta combines the information of AssigneableMeta and ImplicitRefCastMeta to be used by
// escape analysis as well as the compiler.
// It describes the reference operations that are applied to the value of an expression, in this order:
//
//  1. the node itself (Op), yielding a value with BaseLevel reference levels
//  2. Derefs implicit dereferences
//  3. Promotions implicit copies into new reference cells
//  4. if BoxAny, the value is boxed into a Variable (any)
//  5. BoxPromotions copies of the boxed any into new reference cells
//
// Level is the reference level of the resulting value.
type RefOpsMeta struct {
	Op            NodeRefOp
	BaseLevel     uint
	Derefs        uint
	Promotions    uint
	BoxAny        bool
	BoxPromotions uint
	Level         uint
}

var _ ast.MetadataAttachment = RefOpsMeta{}

const RefOpsMetaKind ast.MetadataKind = "DDP_RefOpsMeta"

func (RefOpsMeta) Kind() ast.MetadataKind {
	return RefOpsMetaKind
}

func (m RefOpsMeta) String() string {
	return fmt.Sprintf("RefOpsMeta(Op=%s, BaseLevel=%d, Derefs=%d, Promotions=%d, BoxAny=%t, BoxPromotions=%d, Level=%d)",
		m.Op, m.BaseLevel, m.Derefs, m.Promotions, m.BoxAny, m.BoxPromotions, m.Level)
}

// wether the node yields the address of its storage
func (m RefOpsMeta) AddrOf() bool {
	return m.Op == OpAddr
}

// computes Level from the other fields
func (m *RefOpsMeta) updateLevel() {
	if m.BoxAny {
		m.Level = m.BoxPromotions
		return
	}
	m.Level = m.BaseLevel - m.Derefs + m.Promotions
}

// returns the RefOpsMeta of expr, if present
func GetRefOps(expr ast.Expression) (RefOpsMeta, bool) {
	if expr == nil {
		return RefOpsMeta{}, false
	}
	if att, ok := expr.GetMetadataByKind(RefOpsMetaKind); ok {
		return att.(RefOpsMeta), true
	}
	return RefOpsMeta{}, false
}

// AssignKind describes what the target of an AssignStmt means
type AssignKind int

const (
	// store into a local variable or parameter (or a field of it), Root is the variable
	AssignStoreIntoVar AssignKind = iota
	// the target is a reference variable that is rebound to a new reference, Root is the variable
	AssignRebind
	// store into a global variable (or a field of it), Root is the variable
	AssignStoreIntoGlobal
	// store into a list element, which lives in the GC-managed list array
	AssignStoreIntoList
	// store through a reference whose target is unknown
	AssignStoreThrough
	// store a single character into a Text (string indexing)
	AssignStringIndexing
)

func (k AssignKind) String() string {
	switch k {
	case AssignStoreIntoVar:
		return "StoreIntoVar"
	case AssignRebind:
		return "Rebind"
	case AssignStoreIntoGlobal:
		return "StoreIntoGlobal"
	case AssignStoreIntoList:
		return "StoreIntoList"
	case AssignStoreThrough:
		return "StoreThrough"
	case AssignStringIndexing:
		return "StoreChar"
	}
	return fmt.Sprintf("AssignKind(%d)", int(k))
}

// AssignTargetMeta is attached to every AssignStmt
type AssignTargetMeta struct {
	Target AssignKind
	// the variable that is stored into for AssignStoreIntoVar, AssignRebind and AssignStoreIntoGlobal, nil otherwise
	Root *ast.VarDecl
}

var _ ast.MetadataAttachment = AssignTargetMeta{}

const AssignTargetMetaKind ast.MetadataKind = "DDP_AssignTargetMeta"

func (AssignTargetMeta) Kind() ast.MetadataKind {
	return AssignTargetMetaKind
}

func (m AssignTargetMeta) String() string {
	root := "<nil>"
	if m.Root != nil {
		root = m.Root.Name()
	}
	return fmt.Sprintf("AssignTargetMeta(Target=%s, Root=%s)", m.Target, root)
}

// returns the AssignTargetMeta of stmt, if present
func GetAssignTarget(stmt *ast.AssignStmt) (AssignTargetMeta, bool) {
	if stmt == nil {
		return AssignTargetMeta{}, false
	}
	if att, ok := stmt.GetMetadataByKind(AssignTargetMetaKind); ok {
		return att.(AssignTargetMeta), true
	}
	return AssignTargetMeta{}, false
}
