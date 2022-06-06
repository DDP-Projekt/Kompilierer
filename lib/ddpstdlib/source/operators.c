/*
	defines functions for ddp operators
*/
#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
#include "utf8/utf8.h"
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <float.h>

#define MAKE_BYTE_ARRAY_FROM_CHAR(name, c, num_bytes, ch) \
	ch = (uint32_t)c; \
	if (ch <= 127) { \
		name[0] = (char)c; \
		num_bytes = 1; \
	} else if (ch <= 2047) { \
		name[0] = 192 | (char)(c >> 6); \
		name[1] = 128 | ((char)c)&63; \
		num_bytes = 2; \
	} else if (ch > 1114111 || (55296 <= ch && ch <= 57343)) { \
		printf("invalid utf in ddpchar"); \
		exit(1); \
	} else if (ch <= 65535) { \
		name[0] = 224 | (char)(c >> 12); \
		name[1] = 128 | ((char)c >> 6)&63; \
		name[2] = 128 | ((char)c)&63; \
		num_bytes = 3; \
	} else { \
		name[0] = 240 | (char)(c>>18); \
		name[1] = 128 | (char)(c >> 12); \
		name[2] = 128 | ((char)c >> 6)&63; \
		name[3] = 128 | ((char)c)&63; \
	} \
	name[num_bytes] = '\0';

ddpint inbuilt_int_betrag(ddpint i) {
	return llabs(i);
}

ddpfloat inbuilt_float_betrag(ddpfloat f) {
	return fabs(f);
}

ddpint inbuilt_string_length(ddpstring* str) {
	return (ddpint)utf8_strlen(str->str);
}

ddpstring* inbuilt_string_string_verkettet(ddpstring* str1, ddpstring* str2) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_string_string_verkettet: %p", dstr);

	char* string = ALLOCATE(char, str1->cap - 1 + str2->cap); // remove 1 null-terminator
	memcpy(string, str1->str, str1->cap - 1); // don't copy the null-terminator
	memcpy(&string[str1->cap - 1], str2->str, str2->cap);

	dstr->cap = str1->cap - 1 + str2->cap;
	dstr->str = string;
	return dstr;
}

ddpstring* inbuilt_char_string_verkettet(ddpchar c, ddpstring* str) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_char_string_verkettet: %p", dstr);

	char temp[5];
	size_t num_bytes = 0;
	uint32_t ch;
	MAKE_BYTE_ARRAY_FROM_CHAR(temp, c, num_bytes, ch);
	
	char* string = ALLOCATE(char, str->cap + num_bytes);
	memcpy(&string[num_bytes], str->str, str->cap);
	memcpy(string, temp, num_bytes);

	dstr->cap = str->cap + num_bytes;
	dstr->str = string;
	return dstr;
}

ddpstring* inbuilt_string_char_verkettet(ddpstring* str, ddpchar c) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_string_char_verkettet: %p", dstr);

	char temp[5];
	size_t num_bytes = 0;
	uint32_t ch;
	MAKE_BYTE_ARRAY_FROM_CHAR(temp, c, num_bytes, ch);

	char* string = ALLOCATE(char, str->cap + num_bytes);
	memcpy(string, str->str, str->cap - 1); // don't copy the null-terminator
	memcpy(&string[str->cap - 1], temp, num_bytes);
	string[str->cap + num_bytes] = '\0';

	dstr->cap = str->cap + num_bytes;
	dstr->str = string;
	return dstr;
}

ddpstring* inbuilt_char_char_verkettet(ddpchar c1, ddpchar c2) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_char_char_verkettet: %p", dstr);

	char temp1[5];
	size_t num_bytes1 = 0;
	uint32_t ch;
	MAKE_BYTE_ARRAY_FROM_CHAR(temp1, c1, num_bytes1, ch);

	char temp2[5];
	size_t num_bytes2 = 0;
	MAKE_BYTE_ARRAY_FROM_CHAR(temp2, c2, num_bytes2, ch);

	char* string = ALLOCATE(char, num_bytes1 + num_bytes2 + 1);
	memcpy(string, temp1, num_bytes1);
	memcpy(&string[num_bytes1], temp2, num_bytes2);
	string[num_bytes1 + num_bytes2] = '\0';

	dstr->cap = num_bytes1 + num_bytes2 + 1;
	dstr->str = string;
	return dstr;
}

ddpint inbuilt_string_to_int(ddpstring* str) {
	if (str->cap == 0 || str->str[0] == '\0') return 0; // empty string

	return strtoll(str->str, NULL, 10); // cast the copy to int
}

ddpfloat inbuilt_string_to_float(ddpstring* str) {
	if (str->cap == 0 || str->str[0] == '\0') return 0; // empty string

	return strtod(str->str, NULL); // maybe works with comma seperator? We'll see
}

ddpstring* inbuilt_int_to_string(ddpint i) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_int_to_string: %p", dstr);

	char buffer[20];
	int len = sprintf(buffer, "%ld", i);

	char* string = ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	dstr->str = string;
	dstr->cap = len + 1;
	return dstr;
}

ddpstring* inbuilt_float_to_string(ddpfloat f) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_float_to_string: %p", dstr);

	char buffer[50];
	int len = sprintf(buffer, "%g", f);

	char* string = ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	dstr->str = string;
	dstr->cap = len + 1;
	return dstr;
}

ddpstring* inbuilt_bool_to_string(ddpbool b) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_bool_to_string: %p", dstr);

	char* string;

	if (b) {
		dstr->cap = 5;
		string = ALLOCATE(char, 5);
		memcpy(string, "wahr", 5);
	} else {
		dstr->cap = 7;
		string = ALLOCATE(char, 7);
		memcpy(string, "falsch", 7);
	}
	dstr->str = string;
	return dstr;
}

ddpbool inbuilt_string_equal(ddpstring* str1, ddpstring* str2) {
	if (strlen(str1->str) != strlen(str2->str)) return false; // if the length is different, it's a quick false return
	return memcmp(str1->str, str2->str, str1->cap) == 0;
}

#undef MAKE_BYTE_ARRAY_FROM_CHAR