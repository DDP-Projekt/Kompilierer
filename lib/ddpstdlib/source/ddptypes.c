#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
#include "utf8/utf8.h"

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
ddpstring* inbuilt_string_from_constant(char* str) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_string_from_constant: %p", dstr);
	size_t size = strlen(str) + 1;
	char* string = ALLOCATE(char, size); // the char array of the string (plus null terminator)
	// copy the passed char array
	memcpy(string, str, size);

	// set the string fields
	dstr->str = string;
	dstr->cap = size;
	return dstr;
}

// free a ddpstring
void free_string(ddpstring* str) {
	DBGLOG("inbuilt_free_string: %p", str);
	FREE_ARRAY(char, str->str, str->cap); // free the character array
	FREE(ddpstring, str); // free the string pointer
}

// allocate a new ddpstring as copy of str
ddpstring* inbuilt_deep_copy_string(ddpstring* str) {
	DBGLOG("inbuilt_deep_copy_string: %p", str);
	char* cpy = ALLOCATE(char, str->cap); // allocate the char array for the copy
	memcpy(cpy, str->str, str->cap); // copy the chars
	ddpstring* cpystr = ALLOCATE(ddpstring, 1); // alocate the copy string
	// set the fields of the copy
	cpystr->str = cpy;
	cpystr->cap = str->cap;
	return cpystr;
}