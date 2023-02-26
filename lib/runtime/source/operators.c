/*
	defines functions for ddp operators
*/
#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
#include "utf8/utf8.h"
#include <math.h>
#include <float.h>

ddpint _ddp_string_length(ddpstring* str) {
	return (ddpint)utf8_strlen(str->str);
}

ddpchar _ddp_string_index(ddpstring* str, ddpint index) {
	if (index > str->cap || index < 1 || str->cap <= 1) {
		runtime_error(1, "Index außerhalb der Text Länge (Index war %ld, Text Länge war %ld)\n", index, utf8_strlen(str->str));
	}

	size_t i = 0, len = index;
	while (str->str[i] != 0 && len > 1) {
		i += utf8_num_bytes(str->str + i);
		len--;
	}

	if (str->str[i] == 0) {
		runtime_error(1, "Index außerhalb der Text Länge (Index war %ld, Text Länge war %ld)\n", index, utf8_strlen(str->str));
	}

	return utf8_string_to_char(str->str + i);
}

void _ddp_replace_char_in_string(ddpstring* str, ddpchar ch, ddpint index) {
	if (index > str->cap || index < 1 || str->cap <= 1) {
		runtime_error(1, "Index außerhalb der Text Länge (Index war %ld, Text Länge war %ld)\n", index, utf8_strlen(str->str));
	}

	size_t i = 0, len = index;
	while (str->str[i] != 0 && len > 1) {
		i += utf8_num_bytes(str->str + i);
		len--;
	}

	if (str->str[i] == 0) {
		runtime_error(1, "Index außerhalb der Text Länge (Index war %ld, Text Länge war %ld)\n", index, utf8_strlen(str->str));
	}

	size_t oldCharLen = utf8_num_bytes(str->str + i);
	char newChar[5];
	size_t newCharLen = utf8_char_to_string(newChar, ch);

	if (oldCharLen == newCharLen) { // no need for allocations
		memcpy(str->str + i, newChar, newCharLen);
		return;
	} else if (oldCharLen > newCharLen) { // no need for allocations
		memcpy(str->str + i, newChar, newCharLen);
		memmove(str->str + i + newCharLen, str->str + i + oldCharLen, str->cap - i - oldCharLen);
	} else {
		size_t newStrCap = str->cap - oldCharLen + newCharLen;
		char* newStr = ALLOCATE(char, newStrCap);
		memcpy(newStr, str->str, i); // copy everything before the new char
		memcpy(newStr + i, newChar, newCharLen);
		memcpy(newStr + i + newCharLen, str->str + i + oldCharLen, str->cap - i - oldCharLen);
		FREE_ARRAY(char, str->str, str->cap);
		str->cap = newStrCap;
		str->str = newStr;
	}
}

static ddpint clamp(ddpint i, ddpint min, ddpint max) {
	const ddpint t = i < min ? min : i;
	return t > max ? max : t;
}

void _ddp_string_slice(ddpstring* ret, ddpstring* str, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_string_slice: %p, ret: %p", str, ret);
	ret->cap = 1;
	ret->str = ALLOCATE(char, 1);
	ret->str[0] = '\0';

	if (str->cap <= 1)
		return str; // empty string can stay the same

	size_t start_length = utf8_strlen(str->str);
	index1 = clamp(index1, 1, start_length);
	index2 = clamp(index2, 1, start_length);
	if (index2 < index1) {
		runtime_error(1, "Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n", index1, index2);
	}

	index1--, index2--; // ddp indices start at 1, c indices at 0

	size_t i1 = 0, len = 0;
	while (str->str[i1] != 0 && len != index1) { // while not at null terminator && not at index 1
		++len;
		i1 += utf8_indicated_num_bytes(str->str[i1]);
	}

	size_t i2 = i1;
	while (str->str[i2] != 0 && len != index2) { // while not at null terminator && not at index 2
		++len;
		i2 += utf8_indicated_num_bytes(str->str[i2]);
	}

	ret->cap = (i2 - i1) + 2; // + 1 if indices are equal, + 2 because null-terminator
	ret->str = _ddp_reallocate(ret->str, sizeof(char) * 1, ret->cap);
	memcpy(ret->str, str->str + i1, ret->cap - 1);
	ret->str[ret->cap - 1] = '\0';
}

void _ddp_string_string_verkettet(ddpstring* ret, ddpstring* str1, ddpstring* str2) {
	DBGLOG("_ddp_string_string_verkettet: %p, %p, ret: %p", str1, str2, ret);

	ret->cap = str1->cap - 1 + str2->cap;				 // remove 1 null-terminator
	ret->str = _ddp_reallocate(str1->str, str1->cap, ret->cap);	 // reallocate str1
	memcpy(&ret->str[str1->cap - 1], str2->str, str2->cap); // append str2 and overwrite str1's null-terminator

	str1->cap = 0;
	str1->str = NULL;
}

void _ddp_char_string_verkettet(ddpstring* ret, ddpchar c, ddpstring* str) {
	DBGLOG("_ddp_char_string_verkettet: %p, ret: %p", str, ret);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

	ret->cap = str->cap + num_bytes;
	ret->str = _ddp_reallocate(str->str, str->cap, ret->cap);
	memmove(&ret->str[num_bytes], ret->str, str->cap);
	memcpy(ret->str, temp, num_bytes);

	str->cap = 0;
	str->str = NULL;
}

void _ddp_string_char_verkettet(ddpstring* ret, ddpstring* str, ddpchar c) {
	DBGLOG("_ddp_string_char_verkettet: %p, ret: %p", str, ret);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

	ret->cap = str->cap + num_bytes;
	ret->str = _ddp_reallocate(str->str, str->cap, ret->cap);
	memcpy(&ret->str[str->cap - 1], temp, num_bytes);
	ret->str[ret->cap - 1] = '\0';

	str->str = NULL;
	str->cap = 0;
}

ddpint _ddp_string_to_int(ddpstring* str) {
	if (str->cap == 0 || str->str[0] == '\0') return 0; // empty string

	return strtoll(str->str, NULL, 10); // cast the copy to int
}

ddpfloat _ddp_string_to_float(ddpstring* str) {
	if (str->cap == 0 || str->str[0] == '\0') return 0; // empty string

	return strtod(str->str, NULL); // maybe works with comma seperator? We'll see
}

void _ddp_int_to_string(ddpstring* ret, ddpint i) {
	DBGLOG("_ddp_int_to_string: %p", ret);

	char buffer[21];
	int len = sprintf(buffer, "%lld", i);

	char* string = ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	ret->str = string;
	ret->cap = len + 1;
}

void _ddp_float_to_string(ddpstring* ret, ddpfloat f) {
	DBGLOG("_ddp_float_to_string: %p", ret);

	char buffer[50];
	int len = sprintf(buffer, "%.16g", f);

	char* string = ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	ret->str = string;
	ret->cap = len + 1;
}

void _ddp_bool_to_string(ddpstring* ret, ddpbool b) {
	DBGLOG("_ddp_bool_to_string: %p", ret);

	char* string;

	if (b) {
		ret->cap = 5;
		string = ALLOCATE(char, 5);
		memcpy(string, "wahr", 5);
	} else {
		ret->cap = 7;
		string = ALLOCATE(char, 7);
		memcpy(string, "falsch", 7);
	}
	ret->str = string;
}

void _ddp_char_to_string(ddpstring* ret, ddpchar c) {
	DBGLOG("_ddp_bool_to_string: %p", ret);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // invalid utf8, string will be empty
		num_bytes = 0;
	}

	char* string = ALLOCATE(char, num_bytes + 1);
	memcpy(string, temp, num_bytes);
	string[num_bytes] = '\0';

	ret->cap = num_bytes + 1;
	ret->str = string;
}

ddpbool _ddp_string_equal(ddpstring* str1, ddpstring* str2) {
	if (str1 == str2) return true;
	if (strlen(str1->str) != strlen(str2->str)) return false; // if the length is different, it's a quick false return
	return memcmp(str1->str, str2->str, str1->cap) == 0;
}