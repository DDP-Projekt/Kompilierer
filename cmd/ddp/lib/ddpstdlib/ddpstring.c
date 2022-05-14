#include <string.h>
#include "ddpstring.h"
#include "memory.h"
#include "debug.h"

ddpstring* inbuilt_string_from_constant(ddpchar* str, ddpint len) {
	DBGLOG("inbuilt_string_from_constant");
	ddpchar* string = reallocate(NULL, 0, sizeof(ddpchar) * len);
	for (int i = 0; i < len; i++) {
		string[i] = str[i];
	}
	ddpstring* dstr = allocate_ddpstring();
	dstr->str = string;
	dstr->len = len;
	return dstr;
}

ddpstring* inbuilt_deep_copy_string(ddpstring* str) {
	DBGLOG("inbuilt_deep_copy_string");
	ddpchar* cpy =  reallocate(NULL, 0, sizeof(ddpchar) * str->len);
	memcpy(cpy, str->str, sizeof(ddpchar) * str->len);
	ddpstring* cpystr = allocate_ddpstring();
	cpystr->str = cpy;
	cpystr->len = str->len;
	return cpystr;
}