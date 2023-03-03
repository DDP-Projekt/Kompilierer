/*
	declares types and functions to work with ddp types

	the extern marked functions in this file are generated
	by the ddpcompiler and will be present at link time
*/
#ifndef DDP_TYPES_H
#define DDP_TYPES_H

#include "common.h"

// typedefs of primitive ddp types
typedef int64_t ddpint;
typedef double ddpfloat;
typedef bool ddpbool;
typedef int32_t ddpchar; // needs to be 32 bit to hold every possible unicode character

// a ddp string is a null-terminated utf8-encoded byte array
typedef struct {
	char* str; // the byte array
	ddpint cap; // the capacity of the array
} ddpstring;

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
void _ddp_string_from_constant(ddpstring* ret, char* str);
// free a ddpstring
void _ddp_free_string(ddpstring* str);
// allocate a new ddpstring as copy of str
void _ddp_deep_copy_string(ddpstring* ret, ddpstring* str);

typedef struct {
	ddpint* arr; // the element array
	ddpint len; // the length of the array
	ddpint cap; // the capacity of the array
} ddpintlist;

// allocates a ddpintlist with count elements
extern void _ddp_ddpintlist_from_constants(ddpintlist* ret, ddpint count);
// free a ddpintlist
extern void _ddp_free_ddpintlist(ddpintlist* list);
// deep copies list into ret
extern void _ddp_deep_copy_ddpintlist(ddpintlist* ret, ddpintlist* list);

typedef struct {
	ddpfloat* arr; // the element array
	ddpint len; // the length of the array
	ddpint cap; // the capacity of the array
} ddpfloatlist;

// allocates a ddpfloatlist with count elements
extern void _ddp_ddpfloatlist_from_constants(ddpfloatlist* ret, ddpint count);
// free a ddpfloatlist
extern void _ddp_free_ddpfloatlist(ddpfloatlist* list);
// deep copies list into ret
extern void _ddp_deep_copy_ddpfloatlist(ddpfloatlist* ret, ddpfloatlist* list);

typedef struct {
	ddpbool* arr; // the element array
	ddpint len; // the length of the array
	ddpint cap; // the capacity of the array
} ddpboollist;

// allocates a ddpboollist with count elements
extern void _ddp_ddpboollist_from_constants(ddpboollist* ret, ddpint count);
// free a ddpboollist
extern void _ddp_free_ddpboollist(ddpboollist* list);
// deep copies list into ret
extern void _ddp_deep_copy_ddpboollist(ddpboollist* ret, ddpboollist* list);

typedef struct {
	ddpchar* arr; // the element array
	ddpint len; // the length of the array
	ddpint cap; // the capacity of the array
} ddpcharlist;

// allocates a ddpcharlist with count elements
extern void _ddp_ddpcharlist_from_constants(ddpcharlist* ret, ddpint count);
// free a ddpcharlist
extern void _ddp_free_ddpcharlist(ddpcharlist* list);
// deep copies list into ret
extern void _ddp_deep_copy_ddpcharlist(ddpcharlist* ret, ddpcharlist* list);

typedef struct {
	ddpstring* arr; // the element array
	ddpint len; // the length of the array
	ddpint cap; // the capacity of the array
} ddpstringlist;

// allocates a ddpstringlist with count elements
extern void _ddp_ddpstringlist_from_constants(ddpstringlist* ret, ddpint count);
// free a ddpstringlist
extern void _ddp_free_ddpstringlist(ddpstringlist* list);
// deep copies list into ret
extern void _ddp_deep_copy_ddpstringlist(ddpstringlist* ret, ddpstringlist* list);


// useful typedefs to use when interfacing with ddp code

typedef ddpint* ddpintref;
typedef ddpfloat* ddpfloatref;
typedef ddpbool* ddpboolref;
typedef ddpchar* ddpcharref;
typedef ddpstring* ddpstringref;

typedef ddpintlist* ddpintlistref;
typedef ddpfloatlist* ddpfloatlistref;
typedef ddpboollist* ddpboollistref;
typedef ddpcharlist* ddpcharlistref;
typedef ddpstringlist* ddpstringlistref;

#endif // DDP_TYPES_H