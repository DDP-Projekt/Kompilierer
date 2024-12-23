/*
	declares types and functions to work with ddp types

	the extern marked functions in this file are generated
	by the ddpcompiler and will be present at link time
*/
#ifndef DDP_TYPES_H
#define DDP_TYPES_H

#include "common.h"
#include <assert.h>

static_assert(sizeof(void *) == 8, "sizeof(void*) != 8, unexpected errors could occur");

// typedefs of primitive ddp types
typedef int64_t ddpint;
typedef double ddpfloat;
typedef bool ddpbool;
typedef int32_t ddpchar; // needs to be 32 bit to hold every possible unicode character

// a ddp string is a null-terminated utf8-encoded byte array
typedef struct {
	char *str;	// the byte array
	ddpint cap; // the capacity of the array
} ddpstring;

// to be sure it matches the vtable declaration in ir_string_type.go
static_assert(sizeof(ddpstring) == 16, "sizeof(ddpstring) != 16");

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
void ddp_string_from_constant(ddpstring *ret, char *str);
// free a ddpstring
void ddp_free_string(ddpstring *str);
// allocate a new ddpstring as copy of str
void ddp_deep_copy_string(ddpstring *ret, ddpstring *str);
// returns wether the length of str is 0
ddpbool ddp_string_empty(ddpstring *str);
// returns the strlen of str->str or 0 if str is NULL
ddpint ddp_strlen(ddpstring *str);

typedef void (*free_func_ptr)(void *);
typedef void (*deep_copy_func_ptr)(void *, void *);
typedef ddpbool (*equal_func_ptr)(void *, void *);

typedef struct {
	ddpint type_size;
	free_func_ptr free_func;
	deep_copy_func_ptr deep_copy_func;
	equal_func_ptr equal_func;
} vtable;

#define DDP_SMALL_ANY_BUFF_SIZE 16

typedef struct {
	vtable *vtable_ptr;
	union {
		void *value_ptr;
		uint8_t value[DDP_SMALL_ANY_BUFF_SIZE];
	};
} ddpany;

#define DDP_IS_SMALL_ANY(any) ((any)->vtable_ptr->type_size <= 16)
// returns a pointer to the any's value, taking big vs small any into account
#define DDP_ANY_VALUE_PTR(any) \
	DDP_IS_SMALL_ANY(any) ?    \
		&((any)->value) :      \
		(any)->value_ptr

// frees the given any
void ddp_free_any(ddpany *any);
// places a copy of any in ret
void ddp_deep_copy_any(ddpany *ret, ddpany *any);
// compares two any
ddpbool ddp_any_equal(ddpany *any1, ddpany *any2);

typedef struct {
	ddpint *arr; // the element array
	ddpint len;	 // the length of the array
	ddpint cap;	 // the capacity of the array
} ddpintlist;

// allocates a ddpintlist with count elements
extern void ddp_ddpintlist_from_constants(ddpintlist *ret, ddpint count);
// free a ddpintlist
extern void ddp_free_ddpintlist(ddpintlist *list);
// deep copies list into ret
extern void ddp_deep_copy_ddpintlist(ddpintlist *ret, ddpintlist *list);

typedef struct {
	ddpfloat *arr; // the element array
	ddpint len;	   // the length of the array
	ddpint cap;	   // the capacity of the array
} ddpfloatlist;

// allocates a ddpfloatlist with count elements
extern void ddp_ddpfloatlist_from_constants(ddpfloatlist *ret, ddpint count);
// free a ddpfloatlist
extern void ddp_free_ddpfloatlist(ddpfloatlist *list);
// deep copies list into ret
extern void ddp_deep_copy_ddpfloatlist(ddpfloatlist *ret, ddpfloatlist *list);

typedef struct {
	ddpbool *arr; // the element array
	ddpint len;	  // the length of the array
	ddpint cap;	  // the capacity of the array
} ddpboollist;

// allocates a ddpboollist with count elements
extern void ddp_ddpboollist_from_constants(ddpboollist *ret, ddpint count);
// free a ddpboollist
extern void ddp_free_ddpboollist(ddpboollist *list);
// deep copies list into ret
extern void ddp_deep_copy_ddpboollist(ddpboollist *ret, ddpboollist *list);

typedef struct {
	ddpchar *arr; // the element array
	ddpint len;	  // the length of the array
	ddpint cap;	  // the capacity of the array
} ddpcharlist;

// allocates a ddpcharlist with count elements
extern void ddp_ddpcharlist_from_constants(ddpcharlist *ret, ddpint count);
// free a ddpcharlist
extern void ddp_free_ddpcharlist(ddpcharlist *list);
// deep copies list into ret
extern void ddp_deep_copy_ddpcharlist(ddpcharlist *ret, ddpcharlist *list);

typedef struct {
	ddpstring *arr; // the element array
	ddpint len;		// the length of the array
	ddpint cap;		// the capacity of the array
} ddpstringlist;

// allocates a ddpstringlist with count elements
extern void ddp_ddpstringlist_from_constants(ddpstringlist *ret, ddpint count);
// free a ddpstringlist
extern void ddp_free_ddpstringlist(ddpstringlist *list);
// deep copies list into ret
extern void ddp_deep_copy_ddpstringlist(ddpstringlist *ret, ddpstringlist *list);

typedef struct {
	ddpany *arr; // the element array
	ddpint len;	 // the length of the array
	ddpint cap;	 // the capacity of the array
} ddpanylist;

// allocates a ddpanylist with count elements
extern void ddp_ddpanylist_from_constants(ddpanylist *ret, ddpint count);
// free addpanylist
extern void ddp_free_ddpanylist(ddpanylist *list);
// deep copies list into ret
extern void ddp_deep_copy_ddpanylist(ddpanylist *ret, ddpanylist *list);

// useful macros to work with ddp types

#define DDP_GROWTH_FACTOR (1.5)
#define DDP_BASE_CAPACITY (8)
// helper macro to calculate the new capacity of an array
#define DDP_GROW_CAPACITY(capacity) \
	(capacity < DDP_BASE_CAPACITY ? DDP_BASE_CAPACITY : (ddpint)ceil(capacity * DDP_GROWTH_FACTOR))

#define DDP_EMPTY_STRING \
	(ddpstring) {        \
		NULL, 0          \
	}

#define DDP_EMPTY_ANY \
	(ddpany) {        \
		NULL, {       \
			NULL      \
		}             \
	}

#define DDP_EMPTY_LIST(type) \
	(type) {                 \
		NULL, 0, 0           \
	}

// useful typedefs to use when interfacing with ddp code

typedef ddpint *ddpintref;
typedef ddpfloat *ddpfloatref;
typedef ddpbool *ddpboolref;
typedef ddpchar *ddpcharref;
typedef ddpstring *ddpstringref;
typedef ddpany *ddpanyref;

typedef ddpintlist *ddpintlistref;
typedef ddpfloatlist *ddpfloatlistref;
typedef ddpboollist *ddpboollistref;
typedef ddpcharlist *ddpcharlistref;
typedef ddpstringlist *ddpstringlistref;
typedef ddpanylist *ddpanylistref;

#define DDP_INT_FMT "%lld"
#define DDP_FLOAT_FMT "%.16g"
#define DDP_BOOL_FMT "%d"
#define DDP_CHAR_FMT "%d"
#define DDP_STRING_FMT "%s"

#endif // DDP_TYPES_H
