/*
This file defines functions to define
and work with lists.

The "calling convention" for all those functions inside the llir
is a bit special for non-primitive types (strings, lists, structs):

All functions that should return a non-primitive type, take a pointer
to it as first argument and fill that argument.
This means that the caller must allocate the return values properly
beforehand.
*/
package compiler

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// holds the ir-definitions of a ddp-list-type
// for each list-type such a struct
// should be instantiated by a Compiler
// exactly once
type ddpIrListType struct {
	name                        string
	typ                         llvm.Type // the typedef
	elementType                 ddpIrType // the ir-type of the list elements
	vtable                      llvm.Value
	defaultValue                llvm.Value
	fromConstantsIrFun          llvm.Value // the fromConstans ir func
	freeIrFun                   llvm.Value // the free ir func
	deepCopyIrFun               llvm.Value // the deepCopy ir func
	equalsIrFun                 llvm.Value // the equals ir func
	sliceIrFun                  llvm.Value // the clice ir func
	list_list_concat_IrFunc     llvm.Value // the list_list_verkettet ir func
	list_scalar_concat_IrFunc   llvm.Value // the list_scalar_verkettet ir func
	scalar_scalar_concat_IrFunc llvm.Value // the scalar_scalar_verkettet ir func
	scalar_list_concat_IrFunc   llvm.Value // the scalar_list_verkettet ir func
}

var _ ddpIrType = (*ddpIrListType)(nil)

func (t *ddpIrListType) LLType() llvm.Type {
	return t.typ
}

func (t *ddpIrListType) Name() string {
	return t.name
}

func (*ddpIrListType) IsPrimitive() bool {
	return false
}

func (t *ddpIrListType) DefaultValue() llvm.Value {
	return t.defaultValue
}

func (t *ddpIrListType) VTable() llvm.Value {
	return t.vtable
}

func (t *ddpIrListType) FreeFunc() llvm.Value {
	return t.freeIrFun
}

func (t *ddpIrListType) DeepCopyFunc() llvm.Value {
	return t.deepCopyIrFun
}

func (t *ddpIrListType) EqualsFunc() llvm.Value {
	return t.equalsIrFun
}

// defines the struct of a list type
// and all the necessery functions
// from the given elementType and name
// if declarationOnly is true, all functions are only extern declarations
// C-Equivalent:
/*
	struct listtypename {
		ElementType* arr;
		ddpint len;
		ddpint cap;
	}
*/
func (c *compiler) createListType(name string, elementType ddpIrType, declarationOnly bool) *ddpIrListType {
	list := &ddpIrListType{}
	list.name = name
	list.elementType = elementType
	list.typ = c.llctx.StructType([]llvm.Type{c.ptr, c.ddpint, c.ddpint}, false)

	list.fromConstantsIrFun = c.createListFromConstants(list, declarationOnly)
	list.freeIrFun = c.createListFree(list, declarationOnly)
	list.deepCopyIrFun = c.createListDeepCopy(list, declarationOnly)
	list.equalsIrFun = c.createListEquals(list, declarationOnly)
	list.sliceIrFun = c.createListSlice(list, declarationOnly)

	list.list_list_concat_IrFunc,
		list.list_scalar_concat_IrFunc,
		list.scalar_scalar_concat_IrFunc,
		list.scalar_list_concat_IrFunc = c.createListConcats(list, declarationOnly)

	vtable := llvm.AddGlobal(c.llmod, c.vtable_type, name+"_vtable")
	vtable.SetLinkage(llvm.ExternalLinkage)
	vtable.SetVisibility(llvm.DefaultVisibility)

	if !declarationOnly {
		vtable.SetGlobalConstant(true)
		vtable.SetInitializer(llvm.ConstNamedStruct(c.vtable_type, []llvm.Value{
			llvm.ConstInt(c.ddpint, c.getTypeSize(list), false),
			llvm.ConstNull(c.ptr),
			llvm.ConstNull(c.ptr),
			llvm.ConstNull(c.ptr),
		}))
	}

	list.vtable = vtable
	list.defaultValue = llvm.ConstNull(list.typ)

	return list
}

const (
	list_arr_field_index = 0
	list_len_field_index = 1
	list_cap_field_index = 2
)

/*
defines the ddp_x_from_constants function for a listType

ddp_x_from_constants allocates an array big enough to hold count
elements and stores it inside ret->arr.
It also sets ret->len and ret->cap accordingly

signature:
c.void.IrType() ddp_x_from_constants(x* ret, ddpint count)
*/
func (c *compiler) createListFromConstants(listType *ddpIrListType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_"+listType.name+"_from_constants", llvm.FunctionType(c.void, []llvm.Type{c.ptr, c.ddpint}, false), []string{"ret", "count"}, declarationOnly)
	defer c.popBuilder()

	ret, count := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val

	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	cond := llFuncBuilder.CreateICmp(llvm.IntSGT, count, c.zero, "") // count > 0

	// count > 0 ? allocate(sizeof(t) * count) : NULL
	result := c.createTernary(c.ptr, cond,
		func() llvm.Value { return c.allocateArr(listType.elementType.LLType(), count) },
		func() llvm.Value { return c.Null },
	)

	// get pointers to the struct fields
	retArr, retLen, retCap := c.indexStruct(listType.typ, ret, list_arr_field_index), c.indexStruct(listType.typ, ret, list_len_field_index), c.indexStruct(listType.typ, ret, list_cap_field_index)

	// assign the fields
	c.builder().CreateStore(result, retArr)
	c.builder().CreateStore(count, retLen)
	c.builder().CreateStore(count, retCap)

	c.builder().CreateRet(llvm.Value{})

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn, llFuncBuilder)
	return llFuncBuilder.llFn
}

/*
defines the ddp_free_x function for a listType

ddp_free_x frees the array in list->arr.
It does not change list->len or list->cap, as list
should not be used after being freed.

signature:
c.void.IrType() ddp_free_x(x* list)
*/
func (c *compiler) createListFree(listType *ddpIrListType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_free_"+listType.name, llvm.FunctionType(c.void, []llvm.Type{c.ptr}, false), []string{"p"}, declarationOnly)
	defer c.popBuilder()

	list := llFuncBuilder.params[0].val

	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	if !listType.elementType.IsPrimitive() {
		/*
			for (int i = 0; i < list->len; i++) {
				free(list->arr[i]);
			}
		*/
		listArr, listLen := c.loadStructField(listType.typ, list, list_arr_field_index), c.loadStructField(listType.typ, list, list_len_field_index)
		c.createFor(c.zero, c.forDefaultCond(listLen),
			func(index llvm.Value) {
				val := c.indexArray(listType.elementType.LLType(), listArr, index)
				llFuncBuilder.createCall(listType.elementType.FreeFunc(), val)
			},
		)
	}

	listArr, listCap := c.loadStructField(listType.typ, list, list_arr_field_index), c.loadStructField(listType.typ, list, list_cap_field_index)
	c.freeArr(listType.elementType.LLType(), listArr, listCap)

	c.builder().CreateRet(llvm.Value{})

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn, llFuncBuilder)
	return llFuncBuilder.llFn
}

/*
defines the ddp_deep_copy_x function for a listType

ddp_deep_copy_x allocates a new array of the same capacity
as list->cap and copies list->arr in there.
For non-primitive types deepCopies are created

signature:
c.void.IrType() ddp_deep_copy_x(x* ret, x* list)
*/
func (c *compiler) createListDeepCopy(listType *ddpIrListType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_deep_copy_"+listType.name, llvm.FunctionType(c.void, []llvm.Type{c.ptr, c.ptr}, false), []string{"ret", "p"}, declarationOnly)
	defer c.popBuilder()

	ret, list := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val

	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	arrFieldPtr, lenFieldPtr, capFieldPtr := c.indexStruct(listType.typ, ret, list_arr_field_index), c.indexStruct(listType.typ, ret, list_len_field_index), c.indexStruct(listType.typ, ret, list_cap_field_index)
	origArr, origLen, origCap := c.loadStructField(listType.typ, list, list_arr_field_index), c.loadStructField(listType.typ, list, list_len_field_index), c.loadStructField(listType.typ, list, list_cap_field_index)

	ptrs_equal := llFuncBuilder.CreateICmp(llvm.IntEQ, ret, list, "")
	c.createIfElse(ptrs_equal, func() {
		llFuncBuilder.CreateRet(llvm.Value{})
	},
		nil,
	)

	// allocate(sizeof(t) * list->cap)
	arr := c.allocateArr(listType.elementType.LLType(), origCap)

	// primitive types can easily be copied
	if listType.elementType.IsPrimitive() {
		// memcpy(arr, list->arr, list->len)
		c.memcpyArr(listType.elementType.LLType(), arr, origArr, origLen)
	} else { // non-primitive types need to be seperately deep-copied
		/*
			for (int i = 0; i < list->len; i++) {
				ddp_deep_copy(&ret->arr[i], &list->arr[i])
			}
		*/
		c.createFor(c.zero, c.forDefaultCond(origLen),
			func(index llvm.Value) {
				elementPtr := c.indexArray(listType.elementType.LLType(), arr, index)
				llFuncBuilder.createCall(listType.elementType.DeepCopyFunc(), elementPtr, c.indexArray(listType.elementType.LLType(), origArr, index))
			},
		)
	}

	llFuncBuilder.CreateStore(arr, arrFieldPtr)
	llFuncBuilder.CreateStore(origLen, lenFieldPtr)
	llFuncBuilder.CreateStore(origCap, capFieldPtr)

	llFuncBuilder.CreateRet(llvm.Value{})

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn, llFuncBuilder)
	return llFuncBuilder.llFn
}

/*
defines the ddp_x_equal function for a listType

ddp_x_equal checks wether the two lists are equal

signature:
bool ddp_x_equal(x* list1, x* list2)
*/
func (c *compiler) createListEquals(listType *ddpIrListType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_"+listType.name+"_equal", llvm.FunctionType(c.ddpbool, []llvm.Type{c.ptr, c.ptr}, false), []string{"l1", "l2"}, declarationOnly)
	defer c.popBuilder()

	list1, list2 := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val

	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	ptrs_equal := llFuncBuilder.CreateICmp(llvm.IntEQ, list1, list2, "")
	c.createIfElse(ptrs_equal, func() {
		llFuncBuilder.CreateRet(c.True)
	},
		nil,
	)

	// if (list1->len != list2->len) return false;
	list1_len := c.loadStructField(listType.typ, list1, list_len_field_index)
	len_unequal := llFuncBuilder.CreateICmp(llvm.IntNE, list1_len, c.loadStructField(listType.typ, list2, list_len_field_index), "")
	c.createIfElse(len_unequal, func() {
		llFuncBuilder.CreateRet(c.False)
	},
		nil,
	)

	// compare single elements
	// primitive types can easily be compared
	if listType.elementType.IsPrimitive() {
		// return memcmp(list1->arr, list2->arr, sizeof(T) * list1->len) == 0;
		size := llFuncBuilder.CreateMul(c.sizeof(listType.elementType.LLType()), list1_len, "")
		memcmp := c.memcmp(c.loadStructField(listType.typ, list1, list_arr_field_index), c.loadStructField(listType.typ, list2, list_arr_field_index), size)
		llFuncBuilder.CreateRet(llFuncBuilder.CreateICmp(llvm.IntEQ, memcmp, c.False, ""))
	} else { // non-primitive types need to be seperately compared
		/*
			for (int i = 0; i < list1->len; i++) {
				if (!ddp_element_equal(&list1->arr[i], &list2->arr[i]))
					return false;
			}
		*/
		c.createFor(c.zero, c.forDefaultCond(list1_len),
			func(index llvm.Value) {
				list1_arr, list2_arr := c.loadStructField(listType.typ, list1, list_arr_field_index), c.loadStructField(listType.typ, list2, list_arr_field_index)
				list1_at_count, list2_at_count := c.indexArray(listType.elementType.LLType(), list1_arr, index), c.indexArray(listType.elementType.LLType(), list2_arr, index)
				elements_unequal := llFuncBuilder.CreateXor(llFuncBuilder.createCall(listType.elementType.EqualsFunc(), list1_at_count, list2_at_count), c.True, "")

				c.createIfElse(elements_unequal, func() {
					llFuncBuilder.CreateRet(c.False)
				},
					nil,
				)
			},
		)

		llFuncBuilder.CreateRet(c.True)
	}

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn, llFuncBuilder)
	return llFuncBuilder.llFn
}

/*
defines the ddp_x_slice function for a listType

ddp_x_slice slices a list by two indices

signature:
void ddp_x_slice(x* ret, x* list, ddpint index1, ddpint index2)
*/
func (c *compiler) createListSlice(listType *ddpIrListType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_"+listType.name+"_slice", llvm.FunctionType(c.void, []llvm.Type{c.ptr, c.ptr, c.ddpint, c.ddpint}, false), []string{"ret", "l", "i1", "i2"}, declarationOnly)
	defer c.popBuilder()

	ret, list := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val
	index1, index2 := llFuncBuilder.params[2].val, llFuncBuilder.params[3].val

	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	// empty the ret
	llFuncBuilder.CreateStore(c.Null, c.indexStruct(listType.typ, ret, list_arr_field_index))
	llFuncBuilder.CreateStore(c.zero, c.indexStruct(listType.typ, ret, list_len_field_index))
	llFuncBuilder.CreateStore(c.zero, c.indexStruct(listType.typ, ret, list_cap_field_index))

	listLen := c.loadStructField(listType.typ, list, list_len_field_index)

	// if (list->len <= 0) return;
	list_empty := llFuncBuilder.CreateICmp(llvm.IntSLE, listLen, c.zero, "")
	c.createIfElse(list_empty, func() {
		llFuncBuilder.CreateRet(llvm.Value{})
	},
		nil,
	)

	// helper for the clamp function (does what its name suggests)
	clamp := func(val, min, max llvm.Value) llvm.Value {
		temp := c.createTernary(c.ddpint,
			llFuncBuilder.CreateICmp(llvm.IntSLT, val, min, ""),
			func() llvm.Value { return min },
			func() llvm.Value { return val },
		)
		return c.createTernary(c.ddpint,
			llFuncBuilder.CreateICmp(llvm.IntSLT, temp, max, ""),
			func() llvm.Value { return max },
			func() llvm.Value { return temp },
		)
	}

	// clamp the indices to the list bounds
	index1 = clamp(index1, c.newInt(1), listLen)
	index2 = clamp(index2, c.newInt(1), listLen)

	// validate that the indices are valid
	i2_less_i1 := llFuncBuilder.CreateICmp(llvm.IntSLT, index2, index1, "")
	c.createIfElse(i2_less_i1,
		func() {
			c.runtime_error(1, c.slice_error_string, index1, index2)
		},
		nil,
	)

	// sub 1 from the indices because ddp-indices start at 1 not 0
	index1 = llFuncBuilder.CreateSub(index1, c.newInt(1), "")
	index2 = llFuncBuilder.CreateSub(index2, c.newInt(1), "")

	retArrPtr, retLenPtr, retCapPtr := c.indexStruct(listType.typ, ret, list_arr_field_index), c.indexStruct(listType.typ, ret, list_len_field_index), c.indexStruct(listType.typ, ret, list_cap_field_index)

	/*
		ret->len = (index2 - index1) + 1; // + 1 if indices are equal
		ret->cap = GROW_CAPACITY(ret->len);
		ret->arr = ALLOCATE(elementType, ret->cap);
	*/
	new_len := llFuncBuilder.CreateAdd(llFuncBuilder.CreateSub(index2, index1, ""), c.newInt(1), "")
	llFuncBuilder.CreateStore(new_len, retLenPtr)
	llFuncBuilder.CreateStore(c.growCapacity(new_len), retCapPtr)
	llFuncBuilder.CreateStore(c.allocateArr(listType.elementType.LLType(), c.loadStructField(listType.typ, ret, list_cap_field_index)), retArrPtr)

	if listType.elementType.IsPrimitive() {
		// memcpy primitive types
		c.memcpyArr(listType.elementType.LLType(), c.loadStructField(listType.typ, ret, list_arr_field_index), c.indexArray(listType.typ, c.loadStructField(listType.typ, list, list_arr_field_index), index1), new_len)
	} else {
		/*
			size_t j = 0;
			for (size_t i = index1; i <= index2 && i < list->len; i++, j++) {
				ddp_deep_copy_x(&ret->arr[j], list->arr[i]);
			}
		*/

		retArr := c.loadStructField(listType.typ, ret, list_arr_field_index)
		j := c.NewAlloca(c.i64) // the j index variable
		llFuncBuilder.CreateStore(c.zero, j)
		c.createFor(index1,
			// i <= index2 && i < list->len
			func(i llvm.Value) llvm.Value {
				cond1 := llFuncBuilder.CreateICmp(llvm.IntSLE, i, index2, "")
				cond2 := llFuncBuilder.CreateICmp(llvm.IntSLT, i, listLen, "")
				return llFuncBuilder.CreateAnd(cond1, cond2, "")
			},
			// _deep_copy_x(&ret->arr[j], &list->arr[i]); j++
			func(index llvm.Value) {
				listArr := c.loadStructField(listType.typ, list, list_arr_field_index)
				jval := llFuncBuilder.CreateLoad(c.i64, j, "")
				elementPtr := c.indexArray(listType.elementType.LLType(), retArr, jval)
				listElementPtr := c.indexArray(listType.elementType.LLType(), listArr, index)
				llFuncBuilder.createCall(listType.elementType.DeepCopyFunc(), elementPtr, listElementPtr)
				llFuncBuilder.CreateStore(llFuncBuilder.CreateAdd(jval, c.newInt(1), ""), j)
			},
		)
	}

	llFuncBuilder.CreateRet(llvm.Value{})

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn, llFuncBuilder)
	return llFuncBuilder.llFn
}

/*
defines the ddp_x_y_verkettet functions for a listType
and returns them in the order:
list_list, list_scalar, scalar_scalar, scalar_list
*/
func (c *compiler) createListConcats(listType *ddpIrListType, declarationOnly bool) (llvm.Value, llvm.Value, llvm.Value, llvm.Value) {
	// reusable parts for all 4 functions

	// non-primitive types are passed as pointers
	// until I can pass structs correctly
	scal_param_type := listType.elementType.LLType()
	if !listType.elementType.IsPrimitive() {
		scal_param_type = c.ptr
	}

	/*
		list->len = 0;
		list->cap = 0;
		list->arr = NULL;
	*/
	empty_list := func(list llvm.Value) {
		c.builder().CreateStore(c.Null, c.indexStruct(listType.typ, list, list_arr_field_index))
		c.builder().CreateStore(c.zero, c.indexStruct(listType.typ, list, list_len_field_index))
		c.builder().CreateStore(c.zero, c.indexStruct(listType.typ, list, list_cap_field_index))
	}

	/*
		while (ret->cap < ret->len) ret->cap = GROW_CAPACITY(ret->cap);
		ret->arr = ddp_reallocate(list->arr, sizeof(ddpchar) * list->cap, sizeof(ddpchar) * ret->cap);
	*/
	grow_capacity_and_set_arr := func(ret, list llvm.Value) {
		retCapPtr := c.indexStruct(listType.typ, ret, list_cap_field_index)
		// while (ret->cap < ret->len) ret->cap = GROW_CAPACITY(ret->cap);
		c.createWhile(func() llvm.Value {
			return c.builder().CreateICmp(llvm.IntSLT, c.loadStructField(listType.typ, ret, list_cap_field_index), c.loadStructField(listType.typ, ret, list_len_field_index), "")
		}, func() {
			c.builder().CreateStore(c.growCapacity(c.loadStructField(listType.typ, ret, list_cap_field_index)), retCapPtr)
		})

		// ret->arr = ddp_reallocate(list1->arr, sizeof(elementType) * list1->cap, sizeof(elementType) * ret->cap)
		new_arr := c.growArr(listType.elementType.LLType(), c.loadStructField(listType.typ, list, list_arr_field_index), c.loadStructField(listType.typ, list, list_cap_field_index), c.loadStructField(listType.typ, ret, list_cap_field_index))
		c.builder().CreateStore(new_arr, c.indexStruct(listType.typ, ret, list_arr_field_index))
	}

	finish := func() llvm.Value {
		c.builder().CreateRet(llvm.Value{})

		c.insertFunction(c.builder().fnName, nil, c.builder().llFn, c.builder())
		return c.builder().llFn
	}

	// defines the list_list_verkettet function
	list_list_concat := func() llvm.Value {
		llFuncBuilder := c.newBuilder(fmt.Sprintf("ddp_%s_%s_verkettet", listType.name, listType.name), llvm.FunctionType(c.void, []llvm.Type{c.ptr, c.ptr, c.ptr}, false), []string{"ret", "list1", "list2"}, declarationOnly)
		defer c.popBuilder()

		ret, list1, list2 := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val, llFuncBuilder.params[2].val

		if declarationOnly {
			llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			return llFuncBuilder.llFn
		}

		retLenPtr, retCapPtr := c.indexStruct(listType.typ, ret, list_len_field_index), c.indexStruct(listType.typ, ret, list_cap_field_index)
		// ret->len = list1->len + list2->len
		llFuncBuilder.CreateStore(llFuncBuilder.CreateAdd(c.loadStructField(listType.typ, list1, list_len_field_index), c.loadStructField(listType.typ, list2, list_len_field_index), ""), retLenPtr)
		// ret->cap = list1->cap
		llFuncBuilder.CreateStore(c.loadStructField(listType.typ, list1, list_cap_field_index), retCapPtr)

		grow_capacity_and_set_arr(ret, list1)

		new_arr := c.loadStructField(listType.typ, ret, list_arr_field_index)

		if listType.elementType.IsPrimitive() {
			// memcpy(&ret->arr[list1->len], list2->arr, sizeof(elementType) * list2->len)
			c.memcpyArr(listType.elementType.LLType(), c.indexArray(listType.elementType.LLType(), new_arr, c.loadStructField(listType.typ, list1, list_len_field_index)), c.loadStructField(listType.typ, list2, list_arr_field_index), c.loadStructField(listType.typ, list2, list_len_field_index))
		} else {
			list1Len, list2Len := c.loadStructField(listType.typ, list1, list_len_field_index), c.loadStructField(listType.typ, list2, list_len_field_index)
			list2Arr := c.loadStructField(listType.typ, list2, list_arr_field_index)
			/*
				for (int i = 0; i < list->len; i++) {
					ddp_deep_copy(&ret->arr[i+list1->len], &list2->arr[i])
				}
			*/
			c.createFor(c.zero, c.forDefaultCond(list2Len),
				func(index llvm.Value) {
					elementPtr := c.indexArray(listType.elementType.LLType(), new_arr, llFuncBuilder.CreateAdd(list1Len, index, ""))
					llFuncBuilder.createCall(listType.elementType.DeepCopyFunc(), elementPtr, c.indexArray(listType.elementType.LLType(), list2Arr, index))
				},
			)
		}

		empty_list(list1)

		return finish()
	}

	list_scalar_concat := func() llvm.Value {
		llFuncBuilder := c.newBuilder(fmt.Sprintf("ddp_%s_%s_verkettet", listType.name, listType.elementType.Name()), llvm.FunctionType(c.void, []llvm.Type{c.ptr, c.ptr, scal_param_type}, false), []string{"ret", "list", "scal"}, declarationOnly)
		defer c.popBuilder()

		ret, list, scal := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val, llFuncBuilder.params[2].val

		if declarationOnly {
			llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			return llFuncBuilder.llFn
		}

		retLenPtr, retCapPtr := c.indexStruct(listType.typ, ret, list_len_field_index), c.indexStruct(listType.typ, ret, list_cap_field_index)
		// ret->len = list->len + 1
		c.builder().CreateStore(c.builder().CreateAdd(c.loadStructField(listType.typ, list, list_len_field_index), c.newInt(1), ""), retLenPtr)
		// ret->cap = list->cap
		c.builder().CreateStore(c.loadStructField(listType.typ, list, list_cap_field_index), retCapPtr)

		grow_capacity_and_set_arr(ret, list)

		retArr := c.loadStructField(listType.typ, ret, list_arr_field_index)
		listLen := c.loadStructField(listType.typ, list, list_len_field_index)
		if listType.elementType.IsPrimitive() {
			// ret->arr[list->len] = scal
			c.builder().CreateStore(scal, c.indexArray(listType.elementType.LLType(), retArr, listLen))
		} else {
			// ddp_deep_copy_scal(&ret->arr[list->len], scal)
			dst := c.indexArray(listType.elementType.LLType(), retArr, listLen)
			c.builder().createCall(listType.elementType.DeepCopyFunc(), dst, scal)
		}

		empty_list(list)

		return finish()
	}

	scalar_scalar_concat := func() llvm.Value {
		// string concatenations result in a new string
		if listType.elementType == c.ddpstring {
			return llvm.Value{}
		}

		llFuncBuilder := c.newBuilder(fmt.Sprintf("ddp_%s_%s_verkettet", listType.elementType.Name(), listType.elementType.Name()), llvm.FunctionType(c.void, []llvm.Type{c.ptr, scal_param_type, scal_param_type}, false), []string{"ret", "scal1", "scal2"}, declarationOnly)
		defer c.popBuilder()

		ret, scal1, scal2 := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val, llFuncBuilder.params[2].val

		if declarationOnly {
			llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			return llFuncBuilder.llFn
		}
		retArrPtr, retLenPtr, retCapPtr := c.indexStruct(listType.typ, ret, list_arr_field_index), c.indexStruct(listType.typ, ret, list_len_field_index), c.indexStruct(listType.typ, ret, list_cap_field_index)
		// ret->len = list->len + 1
		c.builder().CreateStore(c.newInt(2), retLenPtr)
		// ret->cap = list->cap
		c.builder().CreateStore(c.growCapacity(c.newInt(2)), retCapPtr)
		// ret->arr = ALLOCATE(elementType, 2)
		c.builder().CreateStore(c.allocateArr(listType.elementType.LLType(), c.builder().CreateLoad(c.ddpint, retCapPtr, "")), retArrPtr)

		retArr := c.loadStructField(listType.typ, ret, list_arr_field_index)
		retArr0Ptr, retArr1Ptr := c.indexArray(listType.elementType.LLType(), retArr, c.zero), c.indexArray(listType.elementType.LLType(), retArr, c.newInt(1))
		if listType.elementType.IsPrimitive() {
			// ret->arr[0] = scal1;
			// ret->arr[1] = scal1;
			c.builder().CreateStore(scal1, retArr0Ptr)
			c.builder().CreateStore(scal2, retArr1Ptr)
		} else {
			// ddp_deep_copy_scalar(&ret->arr[0], scal1);
			// ddp_deep_copy_scalar(&ret->arr[1], scal2);
			llFuncBuilder.createCall(listType.elementType.DeepCopyFunc(), retArr0Ptr, scal1)
			llFuncBuilder.createCall(listType.elementType.DeepCopyFunc(), retArr1Ptr, scal2)
		}

		return finish()
	}

	scalar_list_concat := func() llvm.Value {
		llFuncBuilder := c.newBuilder(fmt.Sprintf("ddp_%s_%s_verkettet", listType.elementType.Name(), listType.name), llvm.FunctionType(c.void, []llvm.Type{c.ptr, scal_param_type, c.ptr}, false), []string{"ret", "scal", "list"}, declarationOnly)
		defer c.popBuilder()

		ret, scal, list := llFuncBuilder.params[0].val, llFuncBuilder.params[1].val, llFuncBuilder.params[2].val

		if declarationOnly {
			llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
			return llFuncBuilder.llFn
		}

		retLenPtr, retCapPtr := c.indexStruct(listType.typ, ret, list_len_field_index), c.indexStruct(listType.typ, ret, list_cap_field_index)
		// ret->len = list->len + 1
		c.builder().CreateStore(c.builder().CreateAdd(c.loadStructField(listType.typ, list, list_len_field_index), c.newInt(1), ""), retLenPtr)
		// ret->cap = list->cap
		c.builder().CreateStore(c.loadStructField(listType.typ, list, list_cap_field_index), retCapPtr)

		grow_capacity_and_set_arr(ret, list)

		// memmove(&ret->arr[1], ret->arr, sizeof(elementType) * list->len);
		retArr := c.loadStructField(listType.typ, ret, list_arr_field_index)
		c.memmoveArr(listType.elementType.LLType(), c.indexArray(listType.elementType.LLType(), retArr, c.newInt(1)), retArr, c.loadStructField(listType.typ, list, list_len_field_index))

		if listType.elementType.IsPrimitive() {
			// ret->arr[0] = scal;
			c.builder().CreateStore(scal, retArr)
		} else {
			// ddp_deep_copy(&ret->arr[0], scal)
			llFuncBuilder.createCall(listType.elementType.DeepCopyFunc(), retArr, scal)
		}

		empty_list(list)

		return finish()
	}

	return list_list_concat(), list_scalar_concat(), scalar_scalar_concat(), scalar_list_concat()
}
