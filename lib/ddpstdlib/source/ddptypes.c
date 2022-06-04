#include <string.h>
#include <stdlib.h>
#include "ddptypes.h"
#include "memory.h"
#include "debug.h"

// allocate and create a ddpstring from a constant ddpchar array
ddpstring* inbuilt_string_from_constant(ddpchar* str, ddpint len) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_string_from_constant: %p", dstr);
	ddpchar* string = ALLOCATE(ddpchar, len); // the char array of the string
	// copy the passed char array
	for (int i = 0; i < len; i++) {
		string[i] = str[i];
	}
	// set the string fields
	dstr->str = string;
	dstr->len = len;
	return dstr;
}

// free a ddpstring
void free_string(ddpstring* str) {
	DBGLOG("inbuilt_free_string: %p", str);
	FREE_ARRAY(ddpchar, str->str, str->len); // free the character array
	FREE(ddpstring, str); // free the string pointer
}

// allocate a new ddpstring as copy of str
ddpstring* inbuilt_deep_copy_string(ddpstring* str) {
	DBGLOG("inbuilt_deep_copy_string: %p", str);
	ddpchar* cpy = ALLOCATE(ddpchar, str->len); // allocate the char array for the copy
	memcpy(cpy, str->str, sizeof(ddpchar) * str->len); // copy the chars
	ddpstring* cpystr = ALLOCATE(ddpstring, 1); // alocate the copy string
	// set the fields of the copy
	cpystr->str = cpy;
	cpystr->len = str->len;
	return cpystr;
}