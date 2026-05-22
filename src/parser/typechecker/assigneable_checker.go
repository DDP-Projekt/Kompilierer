package typechecker

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

type AssigneableMeta struct{}

const AssigneableMetaKind = "AssigneableMeta"

func (AssigneableMeta) Kind() ast.MetadataKind { return AssigneableMetaKind }
func (AssigneableMeta) String() string         { return AssigneableMetaKind }

// checks if the given expression can be assigned to
func checkMarkAssigneable(expr ast.Expression, attachMeta, markTopLevel bool) bool {
	// downstreamAttachMeta := attachMeta && !onlyMarkToplevel
	downstreamAttachMeta := attachMeta

	switch ass := expr.(type) {
	case *ast.Ident:
		_, isConst := ass.Declaration.(*ast.ConstDecl)
		if !isConst && attachMeta && (!ddptypes.IsReference(ass.Type()) || markTopLevel) {
			ass.SetMetadataAttachement(AssigneableMeta{})
		}
		return !isConst
	case *ast.Grouping:
		return checkMarkAssigneable(ass.Expr, downstreamAttachMeta, markTopLevel)
	case *ast.BinaryExpr:
		isAss := isBinaryExprAssignable(ass, attachMeta)
		if isAss && attachMeta && (!ddptypes.IsReference(ass.Type()) || markTopLevel) {
			ass.SetMetadataAttachement(AssigneableMeta{})
		}
		return isAss
	case *ast.CastExpr:
		if ass.OverloadedBy != nil {
			return ddptypes.IsReference(ass.OverloadedBy.Call.Func.ReturnType)
		}

		return checkMarkAssigneable(ass.Lhs, downstreamAttachMeta, false)
	case *ast.UnaryExpr:
		if ass.OverloadedBy != nil {
			return ddptypes.IsReference(ass.OverloadedBy.Call.Func.ReturnType)
		}

		switch ass.Operator {
		case ast.UN_DEREF:
			ass.SetMetadataAttachement(AssigneableMeta{})
			return checkMarkAssigneable(ass.Rhs, downstreamAttachMeta, false)
		}
	}

	return ddptypes.IsReference(expr.Type())
}

func isBinaryExprAssignable(expr *ast.BinaryExpr, attachMeta bool) bool {
	if expr.OverloadedBy != nil {
		return ddptypes.IsReference(expr.OverloadedBy.Call.Func.ReturnType)
	}

	// downstreamAttachMeta := attachMeta && !onlyMarkToplevel
	downstreamAttachMeta := attachMeta

	switch expr.Operator {
	case ast.BIN_FIELD_ACCESS:
		return checkMarkAssigneable(expr.Rhs, downstreamAttachMeta, false)
	case ast.BIN_INDEX:
		return checkMarkAssigneable(expr.Lhs, downstreamAttachMeta, false)
	}
	return false
}

func IsAssignable(expr ast.Expression) bool {
	return checkMarkAssigneable(expr, false, false)
}

func markAssigneable(expr ast.Expression, srcType ddptypes.Type) {
	checkMarkAssigneable(expr, !ddptypes.IsDirectReferenceTo(expr.Type(), srcType), ddptypes.Equal(expr.Type(), srcType))
}
