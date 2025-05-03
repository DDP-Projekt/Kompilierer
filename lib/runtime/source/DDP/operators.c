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

ddpint ddp_string_length(const ddpstring *str) {
	if (ddp_string_empty(str)) {
		return 0;
	}
	return (ddpint)utf8_strlen(DDP_STRING_DATA(str));
}

ddpchar ddp_string_index(const ddpstring *str, ddpint index) {
	if (index < 1) {
		ddp_runtime_error(1, "Texte fangen bei Index 1 an. Es wurde wurde versucht " DDP_INT_FMT " zu indizieren\n", index);
	}

	const char *data = DDP_STRING_DATA(str);

	if (index > str->len || str->len < 1) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(data));
	}

	size_t i = 0, len = index;
	while (data[i] != 0 && len > 1) {
		i += utf8_num_bytes(data + i);
		len--;
	}

	if (&data[i] >= &data[str->len]) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(data));
	}

	uint32_t result;
	utf8_string_to_char(&data[i], &result);
	return (ddpchar)result;
}

void ddp_replace_char_in_string(ddpstring *str, ddpchar ch, ddpint index) {
	if (index < 1) {
		ddp_runtime_error(1, "Texte fangen bei Index 1 an. Es wurde wurde versucht " DDP_INT_FMT " zu indizieren\n", index);
	}

	char *data = DDP_STRING_DATA(str);

	if (index > str->len || str->len < 1) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(data));
	}

	size_t i = 0, len = index;
	while (data != 0 && len > 1) {
		i += utf8_num_bytes(&data[i]);
		len--;
	}

	if (&data[i] >= &data[str->len]) {
		ddp_runtime_error(1, "Index außerhalb der Text Länge (Index war " DDP_INT_FMT ", Text Länge war " DDP_INT_FMT ")\n", index, utf8_strlen(data));
	}

	const size_t oldCharLen = utf8_num_bytes(&data[i]);
	char newChar[5];
	const size_t newCharLen = utf8_char_to_string(newChar, ch);

	if (oldCharLen == newCharLen) { // no length/cap changes
		memcpy(&data[i], newChar, newCharLen);
		return;
	} else if (oldCharLen > newCharLen) { // only changes to length
		memcpy(&data[i], newChar, newCharLen);
		memmove(&data[i + newCharLen], &data[i + oldCharLen], str->len + 1 - i - oldCharLen);
		str->len = str->len - oldCharLen + newCharLen;
		return;
	} else {
		const size_t oldCap = DDP_STRING_CAP(str);
		size_t newStrCap = str->len + 1 - oldCharLen + newCharLen;

		// capacity is sufficient
		if (newStrCap <= oldCap) {
			memmove(&data[i + newCharLen], &data[i + oldCharLen], str->len + 1 - i - oldCharLen); // move old data after new char
			memcpy(&data[i], newChar, newCharLen);												  // copy new char
			str->len = newStrCap - 1;
			return;
		}

		// we need to allocate

		// small becomes large
		if (newStrCap > DDP_SMALL_STRING_BUFF_SIZE) {
			newStrCap = DDP_MAX(newStrCap, DDP_SMALL_ALLOCATION_SIZE);
			char *newStr = DDP_ALLOCATE(char, newStrCap);
			memcpy(newStr, data, i); // copy everything before the new char
			memcpy(&newStr[i], newChar, newCharLen);
			memcpy(&newStr[+i + newCharLen], &data[i + oldCharLen], str->len + 1 - i - oldCharLen);
			str->len = str->len - oldCharLen + newCharLen;
			return;
		}

		// large stays large
		data = ddp_reallocate(data, oldCap, newStrCap);

		memmove(&data[i + newCharLen], &data[i + oldCharLen], str->len + 1 - i - oldCharLen); // move old data after new char
		memcpy(&data[i], newChar, newCharLen);												  // copy new char
		str->len = newStrCap - 1;
		str->large.cap = newStrCap;
		str->large.str = data;
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

	char *data = DDP_STRING_DATA(str);

	size_t start_length = utf8_strlen(data);
	index1 = clamp(index1, 1, start_length);
	index2 = clamp(index2, 1, start_length);
	if (index2 < index1) {
		ddp_runtime_error(1, "Invalide Indexe (Index 1 war " DDP_INT_FMT ", Index 2 war " DDP_INT_FMT ")\n", index1, index2);
	}

	index1--, index2--; // ddp indices start at 1, c indices at 0

	ddpint i1 = 0, len = 0;
	while (data[i1] != 0 && len != index1) { // while not at null terminator && not at index 1
		++len;
		i1 += utf8_indicated_num_bytes(data[i1]);
	}

	ddpint i2 = i1;
	while (data[i2] != 0 && len != index2) { // while not at null terminator && not at index 2
		++len;
		i2 += utf8_indicated_num_bytes(data[i2]);
	}

	const size_t newCap = (i2 - i1) + 1 + utf8_num_bytes(&data[i2]); // + 1 because of the null-terminator, + i2 because indices are inclusive

	const size_t oldTerminatorIndex = i1 + newCap - 1;
	const char oldTerminator = data[oldTerminatorIndex];
	data[oldTerminatorIndex] = '\0';
	ddp_string_from_constant(ret, &data[i1]);
	data[oldTerminatorIndex] = oldTerminator;
}

// concatenate two strings
// guarantees that any memory allocated by str1 is either claimed for the result or freed
void ddp_string_string_verkettet(ddpstring *ret, ddpstring *str1, ddpstring *str2) {
	DDP_DBGLOG("_ddp_string_string_verkettet: %p, %p, ret: %p", str1, str2, ret);

	if (ddp_string_empty(str1) && ddp_string_empty(str2)) {
		*ret = DDP_EMPTY_STRING;
		return;
	} else if (ddp_string_empty(str1)) {
		ddp_deep_copy_string(ret, str2);
		ddp_free_string(str1);
		return;
	} else if (ddp_string_empty(str2)) {
		*ret = *str1;
		*str1 = DDP_EMPTY_STRING;
		return;
	}

	ddp_strncat(str1, DDP_STRING_DATA(str2), str2->len);
	*ret = *str1;
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

	const ddpint oldCap = DDP_STRING_CAP(str);
	const ddpint newLen = num_bytes + str->len;
	if (oldCap >= newLen + 1) {
		char *data = DDP_STRING_DATA(str);
		memmove(&data[num_bytes], data, str->len + 1);
		memcpy(data, temp, num_bytes);
		DDP_DBGLOG("verkettet: %s", data);
		str->len = newLen;
		*ret = *str;
		*str = DDP_EMPTY_STRING;
		return;
	}

	const ddpint newCap = newLen + 1;
	// not enough capacity, so reallocate large string
	str->large.str = ddp_reallocate(str->large.str, str->large.cap, newCap);
	str->large.cap = newCap;

	char *data = DDP_STRING_DATA(str);
	memmove(&data[num_bytes], data, str->len + 1);
	memcpy(data, temp, num_bytes);
	str->len = newLen;
	*ret = *str;
	*str = DDP_EMPTY_STRING;
}

// concatenate a string to a char
// guarantees that any memory allocated by str is either claimed for the result or freed
void ddp_string_char_verkettet(ddpstring *ret, ddpstring *str, ddpchar c) {
	DDP_DBGLOG("_ddp_string_char_verkettet: %p, ret: %p", str, ret);

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

	ddp_strncat(str, temp, num_bytes);
	*ret = *str;
	*str = DDP_EMPTY_STRING;
}

ddpint ddp_string_to_int(ddpstring *str) {
	DDP_DBGLOG("ddp_string_to_int: %s", DDP_STRING_DATA(str));
	if (ddp_string_empty(str)) {
		return 0; // empty string
	}

	return strtoll(DDP_STRING_DATA(str), NULL, 10); // cast the copy to int
}

ddpfloat ddp_string_to_float(ddpstring *str) {
	DDP_DBGLOG("ddp_string_to_float: %s", DDP_STRING_DATA(str));
	if (ddp_string_empty(str)) {
		return 0; // empty string
	}

	return strtod(DDP_STRING_DATA(str), NULL); // maybe works with comma seperator? We'll see
}

void ddp_int_to_string(ddpstring *ret, ddpint i) {
	DDP_DBGLOG("_ddp_int_to_string: %p", ret);

	char buffer[22];
	sprintf(buffer, DDP_INT_FMT, i);

	ddp_string_from_constant(ret, buffer);
}

void ddp_float_to_string(ddpstring *ret, ddpfloat f) {
	DDP_DBGLOG("_ddp_float_to_string: %p", ret);

	char buffer[51];
	sprintf(buffer, DDP_FLOAT_FMT, f);

	ddp_string_from_constant(ret, buffer);
}

void ddp_bool_to_string(ddpstring *ret, ddpbool b) {
	DDP_DBGLOG("_ddp_bool_to_string: %p", ret);

	if (b) {
		ret->len = 4;
		memcpy(ret->small.str, "wahr", 5);
	} else {
		ret->len = 6;
		memcpy(ret->small.str, "falsch", 7);
	}
}

void ddp_char_to_string(ddpstring *ret, ddpchar c) {
	DDP_DBGLOG("_ddp_bool_to_string: %p", ret);

	size_t num_bytes = utf8_char_to_string(ret->small.str, c);
	if (num_bytes == (size_t)-1) { // invalid utf8, string will be empty
		num_bytes = 0;
	}
	ret->len = num_bytes;
	ret->small.str[num_bytes] = '\0';
}

ddpbool ddp_string_equal(ddpstring *str1, ddpstring *str2) {
	if (str1 == str2) {
		return true;
	}
	if (str1->len != str2->len) {
		return false; // if the length is different, it's a quick false return
	}
	return memcmp(DDP_STRING_DATA(str1), DDP_STRING_DATA(str2), str1->len) == 0;
}
