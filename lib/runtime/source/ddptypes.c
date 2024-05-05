#include "ddptypes.h"
#include "ddpmemory.h"
#include "debug.h"
#include <stdlib.h>
#include <string.h>

// allocate and create a ddpstring from a constant char array
// str must be null-terminated
void ddp_string_from_constant(ddpstring *ret, char *str) {
	DDP_DBGLOG("_ddp_string_from_constant: ret: %p", ret);
	size_t size = strlen(str) + 1;
	if (size == 1) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	char *string = DDP_ALLOCATE(char, size); // the char array of the string (plus null terminator)
	// copy the passed char array
	memcpy(string, str, size);

	// set the string fields
	ret->str = string;
	ret->cap = size;
}

// free a ddpstring
void ddp_free_string(ddpstring *str) {
	DDP_DBGLOG("free_string: %p", str);
	DDP_FREE_ARRAY(char, str->str, str->cap); // free the character array
}

// allocate a new ddpstring as copy of str
void ddp_deep_copy_string(ddpstring *ret, ddpstring *str) {
	DDP_DBGLOG("_ddp_deep_copy_string: %p, ret: %p", str, ret);
	if (ret == str) {
		return;
	}
	char *cpy = DDP_ALLOCATE(char, str->cap); // allocate the char array for the copy
	memcpy(cpy, str->str, str->cap);		  // copy the chars
	// set the fields of the copy
	ret->str = cpy;
	ret->cap = str->cap;
}

// returns wether the length of str is 0
ddpbool ddp_string_empty(ddpstring *str) {
	return str->str == NULL || str->cap <= 0 || (str->str[0] == '\0');
}

ddpint ddp_strlen(ddpstring *str) {
	if (str == NULL || str->str == NULL) {
		return 0;
	}
	return strlen(str->str);
}
