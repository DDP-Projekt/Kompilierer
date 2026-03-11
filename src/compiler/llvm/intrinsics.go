package llvm

/*
#include "llvm-c/Core.h"
#include "llvm-c/Comdat.h"
#include "IRBindings.h"
#include <stdlib.h>
*/
import "C"
import "unsafe"

func LookupIntrinsicID(name string) uint {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return uint(C.LLVMLookupIntrinsicID(cname, C.size_t(len(name))))
}

func (m Module) GetIntrinsicDeclaration(id uint, overloadedTypes []Type) (v Value) {
	ptr, ntypes := llvmTypeRefs(overloadedTypes)
	v.C = C.LLVMGetIntrinsicDeclaration(m.C, C.unsigned(id), ptr, C.size_t(ntypes))
	return v
}
