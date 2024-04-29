/*
	defines functions for ddp operators
*/
#include "ddpmemory.h"
#include "ddptypes.h"
#include "debug.h"
#include "utf8/utf8.h"
#include <float.h>
#include <stdlib.h>
#include <string.h>

ddpint ddp_string_length(ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0;
	}
	return (ddpint)utf8_strlen(str->str);
}

ddpchar ddp_string_index(ddpstring *str, ddpint index) {
	if (index > str->cap || index < 1 || str->cap <= 1) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str->str));
	}

	size_t i = 0, len = index;
	while (str->str[i] != 0 && len > 1) {
		i += utf8_num_bytes(str->str + i);
		len--;
	}

	if (str->str[i] == 0) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str->str));
	}

	return utf8_string_to_char(str->str + i);
}

void ddp_replace_char_in_string(ddpstring *str, ddpchar ch, ddpint index) {
	if (index > str->cap || index < 1 || str->cap <= 1) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str->str));
	}

	size_t i = 0, len = index;
	while (str->str[i] != 0 && len > 1) {
		i += utf8_num_bytes(str->str + i);
		len--;
	}

	if (str->str[i] == 0) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str->str));
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
		char *newStr = DDP_ALLOCATE(char, newStrCap);
		memcpy(newStr, str->str, i); // copy everything before the new char
		memcpy(newStr + i, newChar, newCharLen);
		memcpy(newStr + i + newCharLen, str->str + i + oldCharLen, str->cap - i - oldCharLen);
		DDP_FREE_ARRAY(char, str->str, str->cap);
		str->cap = newStrCap;
		str->str = newStr;
	}
}

static ddpint clamp(ddpint i, ddpint min, ddpint max) {
	const ddpint t = i < min ? min : i;
	return t > max ? max : t;
}

void ddp_string_slice(ddpstring *ret, ddpstring *str, ddpint index1, ddpint index2) {
	DBGLOG("_ddp_string_slice: %p, ret: %p", str, ret);
	*ret = DDP_EMPTY_STRING;

	if (ddp_string_empty(str)) {
		return; // empty string can stay the same
	}

	size_t start_length = utf8_strlen(str->str);
	index1 = clamp(index1, 1, start_length);
	index2 = clamp(index2, 1, start_length);
	if (index2 < index1) {
		ddp_runtime_error(1, "Invalide Indexe (Index 1 war " DDP_INT_FMT ", Index 2 war " DDP_INT_FMT ")\n", index1, index2);
	}

	index1--, index2--; // ddp indices start at 1, c indices at 0

	ddpint i1 = 0, len = 0;
	while (str->str[i1] != 0 && len != index1) { // while not at null terminator && not at index 1
		++len;
		i1 += utf8_indicated_num_bytes(str->str[i1]);
	}

	ddpint i2 = i1;
	while (str->str[i2] != 0 && len != index2) { // while not at null terminator && not at index 2
		++len;
		i2 += utf8_indicated_num_bytes(str->str[i2]);
	}

	ret->cap = (i2 - i1) + 2; // + 1 if indices are equal, + 2 because null-terminator
	ret->str = ddp_reallocate(ret->str, sizeof(char) * 1, ret->cap);
	memcpy(ret->str, str->str + i1, ret->cap - 1);
	ret->str[ret->cap - 1] = '\0';
}

void ddp_string_string_verkettet(ddpstring *ret, ddpstring *str1, ddpstring *str2) {
	DBGLOG("_ddp_string_string_verkettet: %p, %p, ret: %p", str1, str2, ret);

	if (ddp_string_empty(str1) && ddp_string_empty(str2)) {
		*ret = DDP_EMPTY_STRING;
		return;
	} else if (ddp_string_empty(str1)) {
		ddp_deep_copy_string(ret, str2);
		return;
	} else if (ddp_string_empty(str2)) {
		ddp_deep_copy_string(ret, str1);
		return;
	}

	ret->cap = str1->cap - 1 + str2->cap;					   // remove 1 null-terminator
	ret->str = ddp_reallocate(str1->str, str1->cap, ret->cap); // reallocate str1
	memcpy(&ret->str[str1->cap - 1], str2->str, str2->cap);	   // append str2 and overwrite str1's null-terminator

	*str1 = DDP_EMPTY_STRING;
}

void ddp_char_string_verkettet(ddpstring *ret, ddpchar c, ddpstring *str) {
	DBGLOG("_ddp_char_string_verkettet: %p, ret: %p", str, ret);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

	if (ddp_string_empty(str)) {
		ddp_string_from_constant(ret, temp);
		return;
	}

	ret->cap = str->cap + num_bytes;
	ret->str = ddp_reallocate(str->str, str->cap, ret->cap);
	memmove(&ret->str[num_bytes], ret->str, str->cap);
	memcpy(ret->str, temp, num_bytes);

	*str = DDP_EMPTY_STRING;
}

void ddp_string_char_verkettet(ddpstring *ret, ddpstring *str, ddpchar c) {
	DBGLOG("_ddp_string_char_verkettet: %p, ret: %p", str, ret);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

	if (ddp_string_empty(str)) {
		ddp_string_from_constant(ret, temp);
		return;
	}

	ret->cap = str->cap + num_bytes;
	ret->str = ddp_reallocate(str->str, str->cap, ret->cap);
	memcpy(&ret->str[str->cap - 1], temp, num_bytes);
	ret->str[ret->cap - 1] = '\0';

	*str = DDP_EMPTY_STRING;
}

ddpint ddp_string_to_int(ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0; // empty string
	}

	return strtoll(str->str, NULL, 10); // cast the copy to int
}

ddpfloat ddp_string_to_float(ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0; // empty string
	}

	return strtod(str->str, NULL); // maybe works with comma seperator? We'll see
}

void ddp_int_to_string(ddpstring *ret, ddpint i) {
	DBGLOG("_ddp_int_to_string: %p", ret);

	char buffer[21];
	int len = sprintf(buffer, DDP_INT_FMT, i);

	char *string = DDP_ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	ret->str = string;
	ret->cap = len + 1;
}

void ddp_float_to_string(ddpstring *ret, ddpfloat f) {
	DBGLOG("_ddp_float_to_string: %p", ret);

	char buffer[50];
	int len = sprintf(buffer, DDP_FLOAT_FMT, f);

	char *string = DDP_ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	ret->str = string;
	ret->cap = len + 1;
}

void ddp_bool_to_string(ddpstring *ret, ddpbool b) {
	DBGLOG("_ddp_bool_to_string: %p", ret);

	char *string;

	if (b) {
		ret->cap = 5;
		string = DDP_ALLOCATE(char, 5);
		memcpy(string, "wahr", 5);
	} else {
		ret->cap = 7;
		string = DDP_ALLOCATE(char, 7);
		memcpy(string, "falsch", 7);
	}
	ret->str = string;
}

void ddp_char_to_string(ddpstring *ret, ddpchar c) {
	DBGLOG("_ddp_bool_to_string: %p", ret);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // invalid utf8, string will be empty
		num_bytes = 0;
	}

	char *string = DDP_ALLOCATE(char, num_bytes + 1);
	memcpy(string, temp, num_bytes);
	string[num_bytes] = '\0';

	ret->cap = num_bytes + 1;
	ret->str = string;
}

ddpbool ddp_string_equal(ddpstring *str1, ddpstring *str2) {
	if (str1 == str2) {
		return true;
	}
	if (ddp_strlen(str1) != ddp_strlen(str2)) {
		return false; // if the length is different, it's a quick false return
	}
	return memcmp(str1->str, str2->str, str1->cap) == 0;
}
