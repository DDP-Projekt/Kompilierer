/*
	defines functions for ddp operators
*/
#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/debug.h"
#include "DDP/utf8/utf8.h"
#include <float.h>
#include <stdlib.h>
#include <string.h>

ddpint ddp_string_length(ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0;
	}
	return (ddpint)utf8_strlen(DDP_GET_STRING_PTR(str));
}

ddpchar ddp_string_index(ddpstring *str, ddpint index) {
	if (index < 1) {
		ddp_runtime_error(1, "Texte fangen bei Index 1 an. Es wurde wurde versucht " DDP_INT_FMT " zu indizieren\n", index);
	}

	const char *str_ptr = DDP_GET_STRING_PTR(str);
	if (index > str->cap || str->cap <= 1) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str_ptr));
	}

	size_t i = 0, len = index;
	while (str_ptr[i] != 0 && len > 1) {
		i += utf8_num_bytes(str_ptr + i);
		len--;
	}

	if (str_ptr[i] == 0) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str_ptr));
	}

	uint32_t result;
	utf8_string_to_char(str_ptr + i, &result);
	return (ddpchar)result;
}

void ddp_replace_char_in_string(ddpstring *str, ddpchar ch, ddpint index) {
	DDP_DBGLOG("ddp_replace_char_in_string");
	if (index < 1) {
		ddp_runtime_error(1, "Texte fangen bei Index 1 an. Es wurde wurde versucht " DDP_INT_FMT " zu indizieren\n", index);
	}

	char *str_ptr = DDP_GET_STRING_PTR(str);
	if (index > str->cap || str->cap <= 1) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str_ptr));
	}

	size_t i = 0, len = index;
	while (str_ptr[i] != 0 && len > 1) {
		i += utf8_num_bytes(str_ptr + i);
		len--;
	}

	if (str_ptr[i] == 0) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(str_ptr));
	}

	size_t oldCharLen = utf8_num_bytes(str_ptr + i);
	char newChar[5];
	size_t newCharLen = utf8_char_to_string(newChar, ch);

	if (oldCharLen == newCharLen) { // no need for allocations
		memcpy(str_ptr + i, newChar, newCharLen);
		return;
	} else if (oldCharLen > newCharLen) { // no need for allocations
		memcpy(str_ptr + i, newChar, newCharLen);
		memmove(str_ptr + i + newCharLen, str_ptr + i + oldCharLen, str->cap - i - oldCharLen);
	} else {
		size_t newStrCap = str->cap - oldCharLen + newCharLen;

		DDP_DBGLOG("newCharLen > oldCharLen");
		if (newStrCap <= DDP_SMALL_STRING_LIMIT) {
			DDP_DBGLOG("newStrCap <= DDP_SMALL_STRING_LIMIT");
			memmove(str_ptr + i + newCharLen, str_ptr + i + oldCharLen, str->cap - i - oldCharLen);
			memcpy(str_ptr + i, newChar, newCharLen);
			str->cap = newStrCap;
			return;
		}

		if (DDP_IS_SMALL_STRING(str)) {
			char *newStr = DDP_ALLOCATE(char, newStrCap);
			memcpy(newStr, str_ptr, i);
			memcpy(str_ptr + i, newChar, newCharLen);
			memcpy(str_ptr + i + newCharLen, str_ptr + i + oldCharLen, str->cap - i - oldCharLen);

			str->str = newStr;
			str->refc = NULL;
			str->cap = newStrCap;
		} else {
			str_ptr = ddp_reallocate(str->str, str->cap, newStrCap);
			memmove(str_ptr + i + newCharLen, str_ptr + i + oldCharLen, str->cap - i - oldCharLen);
			memcpy(str_ptr + i, newChar, newCharLen);
			str->cap = newStrCap;
		}
	}
}

static ddpint clamp(ddpint i, ddpint min, ddpint max) {
	const ddpint t = i < min ? min : i;
	return t > max ? max : t;
}

void ddp_string_slice(ddpstring *ret, ddpstring *str, ddpint index1, ddpint index2) {
	DDP_DBGLOG("_ddp_string_slice: %p, ret: %p", str, ret);
	*ret = DDP_EMPTY_STRING;

	if (ddp_string_empty(str)) {
		return; // empty string can stay the same
	}

	char *str_ptr = DDP_GET_STRING_PTR(str);
	size_t start_length = utf8_strlen(str_ptr);
	index1 = clamp(index1, 1, start_length);
	index2 = clamp(index2, 1, start_length);
	if (index2 < index1) {
		ddp_runtime_error(1, "Invalide Indexe (Index 1 war " DDP_INT_FMT ", Index 2 war " DDP_INT_FMT ")\n", index1, index2);
	}

	index1--, index2--; // ddp indices start at 1, c indices at 0

	ddpint i1 = 0, len = 0;
	while (str_ptr[i1] != 0 && len != index1) { // while not at null terminator && not at index 1
		++len;
		i1 += utf8_indicated_num_bytes(str_ptr[i1]);
	}

	ddpint i2 = i1;
	while (str_ptr[i2] != 0 && len != index2) { // while not at null terminator && not at index 2
		++len;
		i2 += utf8_indicated_num_bytes(str_ptr[i2]);
	}

	ret->refc = NULL;
	ret->cap = (i2 - i1) + 1;				  // + 1 because null-terminator
	ret->cap += utf8_num_bytes(str_ptr + i2); // + 1 because indices are inclusive

	char *ret_ptr = DDP_GET_STRING_PTR(ret);
	if (DDP_IS_SMALL_STRING(ret)) {
		memcpy(ret_ptr, str_ptr + i1, ret->cap - 1);
		ret_ptr[ret->cap - 1] = '\0';
	} else {
		ret->str = ddp_reallocate(ret->str, 0, ret->cap);
		memcpy(ret->str, str_ptr + i1, ret->cap - 1);
		ret->str[ret->cap - 1] = '\0';
	}
}

// concatenate two strings
// guarantees that any memory allocated by str1 is either claimed for the result or freed
void ddp_string_string_verkettet(ddpstring *ret, ddpstring *str1, ddpstring *str2) {
	DDP_DBGLOG("ddp_string_string_verkettet: %p, %p, ret: %p", str1, str2, ret);

	if (ddp_string_empty(str1) && ddp_string_empty(str2)) {
		*ret = DDP_EMPTY_STRING;
		ddp_free_string(str1);
		return;
	} else if (ddp_string_empty(str1)) {
		ddp_shallow_copy_string(ret, str2);
		ddp_free_string(str1);
		return;
	} else if (ddp_string_empty(str2)) {
		*ret = *str1;
		*str1 = DDP_EMPTY_STRING;
		return;
	}

	ret->cap = str1->cap - 1 + str2->cap; // remove 1 null-terminator
	if (DDP_IS_SMALL_STRING(ret)) {
		char *ret_ptr = DDP_GET_STRING_PTR(ret);
		memcpy(ret_ptr, DDP_GET_STRING_PTR(str1), str1->cap - 1);
		memcpy(&ret_ptr[str1->cap - 1], DDP_GET_STRING_PTR(str2), str2->cap);
		ddp_free_string(str1);
		return;
	} else {
		DDP_DBGLOG("concatinating new large string");
		ret->refc = NULL;
		if (DDP_IS_SMALL_STRING(str1)) {
			ret->str = DDP_ALLOCATE(char, ret->cap);
			DDP_DBGLOG("copying str1");
			memcpy(ret->str, DDP_GET_STRING_PTR(str1), str1->cap - 1);
		} else {
			ret->str = ddp_reallocate(str1->str, str1->cap, ret->cap);
		}

		memcpy(&ret->str[str1->cap - 1], DDP_GET_STRING_PTR(str2), str2->cap); // append str2 and overwrite str1's null-terminator
	}
	*str1 = DDP_EMPTY_STRING;
}

// concatenate a char to a string
// guarantees that any memory allocated by str is either claimed for the result or freed
void ddp_char_string_verkettet(ddpstring *ret, ddpchar c, ddpstring *str) {
	DDP_DBGLOG("_ddp_char_string_verkettet: %p, ret: %p", str, ret);

	char temp[5];
	size_t num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == (size_t)-1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

	if (ddp_string_empty(str)) {
		ddp_free_string(str);
		ddp_string_from_constant(ret, temp);
		return;
	}

	ret->cap = str->cap + num_bytes;
	if (DDP_IS_SMALL_STRING(ret)) {
		char *ret_ptr = DDP_GET_STRING_PTR(ret);
		memcpy(ret_ptr, temp, num_bytes);
		memcpy(ret_ptr + num_bytes, DDP_GET_STRING_PTR(str), str->cap);
	} else if DDP_IS_SMALL_STRING (str) {
		ret->refc = NULL;
		ret->str = DDP_ALLOCATE(char, ret->cap);
		memcpy(ret->str, temp, num_bytes);
		memcpy(ret->str + num_bytes, DDP_GET_STRING_PTR(str), str->cap);
	} else {
		ret->refc = NULL;
		ret->str = ddp_reallocate(str->str, str->cap, ret->cap);
		memmove(&ret->str[num_bytes], ret->str, str->cap);
		memcpy(ret->str, temp, num_bytes);
		*str = DDP_EMPTY_STRING;
	}
}

// concatenate a string to a char
// guarantees that any memory allocated by str is either claimed for the result or freed
void ddp_string_char_verkettet(ddpstring *ret, ddpstring *str, ddpchar c) {
	DDP_DBGLOG("_ddp_string_char_verkettet: %p, ret: %p", str, ret);

	char temp[5];
	size_t num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == (size_t)-1) { // if c is invalid utf8, we return simply a copy of str
		*ret = *str;
		*str = DDP_EMPTY_STRING;
		return;
	}

	if (ddp_string_empty(str)) {
		ddp_free_string(str);
		ddp_string_from_constant(ret, temp);
		return;
	}

	ret->cap = str->cap + num_bytes;
	if (DDP_IS_SMALL_STRING(ret)) {
		char *ret_ptr = DDP_GET_STRING_PTR(ret);
		memcpy(ret_ptr, DDP_GET_STRING_PTR(str), str->cap - 1);
		memcpy(ret_ptr + str->cap - 1, temp, num_bytes);
		ret_ptr[ret->cap - 1] = '\0';
		ddp_free_string(str);
	} else if (DDP_IS_SMALL_STRING(str)) {
		ret->refc = NULL;
		ret->str = DDP_ALLOCATE(char, ret->cap);
		memcpy(ret->str, DDP_GET_STRING_PTR(str), str->cap - 1);
		memcpy(ret->str + str->cap - 1, temp, num_bytes);
		ret->str[ret->cap - 1] = '\0';
		ddp_free_string(str);
	} else {
		ret->refc = NULL;
		ret->str = ddp_reallocate(str->str, str->cap, ret->cap);
		memcpy(&ret->str[str->cap - 1], temp, num_bytes);
		ret->str[ret->cap - 1] = '\0';
		*str = DDP_EMPTY_STRING;
	}
}

ddpint ddp_string_to_int(ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0; // empty string
	}

	return strtoll(DDP_GET_STRING_PTR(str), NULL, 10); // cast the copy to int
}

ddpfloat ddp_string_to_float(ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0; // empty string
	}

	return strtod(DDP_GET_STRING_PTR(str), NULL); // maybe works with comma seperator? We'll see
}

void ddp_int_to_string(ddpstring *ret, ddpint i) {
	DDP_DBGLOG("_ddp_int_to_string: %p", ret);

	char buffer[21];
	int len = sprintf(buffer, DDP_INT_FMT, i);

	ret->cap = len + 1;
	if (DDP_IS_SMALL_STRING(ret)) {
		char *ret_ptr = DDP_GET_STRING_PTR(ret);
		memcpy(ret_ptr, buffer, len);
		ret_ptr[len] = '\0';
	} else {
		ret->refc = NULL;
		ret->str = DDP_ALLOCATE(char, ret->cap); // the char array of the string + null-terminator
		memcpy(ret->str, buffer, len);
		ret->str[len] = '\0';
	}
}

void ddp_float_to_string(ddpstring *ret, ddpfloat f) {
	DDP_DBGLOG("_ddp_float_to_string: %p", ret);

	char buffer[50];
	int len = sprintf(buffer, DDP_FLOAT_FMT, f);

	ret->cap = len + 1;
	if (DDP_IS_SMALL_STRING(ret)) {
		char *ret_ptr = DDP_GET_STRING_PTR(ret);
		memcpy(ret_ptr, buffer, len);
		ret_ptr[len] = '\0';
	} else {
		ret->refc = NULL;
		ret->str = DDP_ALLOCATE(char, ret->cap); // the char array of the string + null-terminator
		memcpy(ret->str, buffer, len);
		ret->str[len] = '\0';
	}
}

void ddp_bool_to_string(ddpstring *ret, ddpbool b) {
	DDP_DBGLOG("_ddp_bool_to_string: %p", ret);

	if (b) {
		ret->cap = 5;
		memcpy(DDP_GET_STRING_PTR(ret), "wahr", 5);
	} else {
		ret->cap = 7;
		memcpy(DDP_GET_STRING_PTR(ret), "falsch", 7);
	}
}

void ddp_char_to_string(ddpstring *ret, ddpchar c) {
	DDP_DBGLOG("_ddp_bool_to_string: %p", ret);

	char temp[5];
	size_t num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == (size_t)-1) { // invalid utf8, string will be empty
		num_bytes = 0;
	}

	ret->cap = num_bytes + 1;
	char *ret_ptr = DDP_GET_STRING_PTR(ret);
	memcpy(ret_ptr, temp, num_bytes);
	ret_ptr[num_bytes] = '\0';
}

ddpbool ddp_string_equal(ddpstring *str1, ddpstring *str2) {
	if (str1 == str2) {
		return true;
	}
	if (str1->cap != str2->cap || ddp_strlen(str1) != ddp_strlen(str2)) {
		return false; // if the length is different, it's a quick false return
	}
	if ((!DDP_IS_SMALL_STRING(str1) && !DDP_IS_SMALL_STRING(str2)) && str1->refc != NULL && str1->refc == str2->refc) {
		return true;
	}
	return memcmp(DDP_GET_STRING_PTR(str1), DDP_GET_STRING_PTR(str2), str1->cap) == 0;
}
