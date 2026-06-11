//go:build amd64

package compiler

import "github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"

func (c *compiler) tag_pointer(p llvm.Value) llvm.Value {
	addr := c.builder().CreatePtrToInt(p, c.i64, "")
	return c.builder().CreateIntToPtr(c.builder().CreateOr(addr, c.tag_bit, ""), c.ptr, "")
}
