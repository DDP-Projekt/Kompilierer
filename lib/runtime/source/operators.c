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

ddpstring* _ddp_string_slice(ddpstring* str, ddpint index1, ddpint index2) {
	ddpstring* new_str = ALLOCATE(ddpstring, 1);
	DBGLOG("_ddp_string_slice: %p", new_str);
	new_str->cap = 1;
	new_str->str = ALLOCATE(char, 1);
	new_str->str[0] = '\0';

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

	new_str->cap = (i2 - i1) + 2; // + 1 if indices are equal, + 2 because null-terminator
	new_str->str = reallocate(new_str->str, sizeof(char) * 1, new_str->cap);
	memcpy(new_str->str, str->str + i1, new_str->cap - 1);
	new_str->str[new_str->cap - 1] = '\0';

	return new_str;
}

ddpstring* _ddp_string_string_verkettet(ddpstring* str1, ddpstring* str2) {
	ddpstring* new_str = ALLOCATE(ddpstring, 1);
	DBGLOG("_ddp_string_string_verkettet: %p", new_str);

	new_str->cap = str1->cap - 1 + str2->cap;				 // remove 1 null-terminator
	new_str->str = reallocate(str1->str, str1->cap, new_str->cap);	 // reallocate str1
	memcpy(&new_str->str[str1->cap - 1], str2->str, str2->cap); // append str2 and overwrite str1's null-terminator

	str1->cap = 0;
	str1->str = NULL;
	return new_str;
}

ddpstring* _ddp_char_string_verkettet(ddpchar c, ddpstring* str) {
	ddpstring* new_str = ALLOCATE(ddpstring, 1);
	DBGLOG("_ddp_char_string_verkettet: %p", new_str);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

	new_str->cap = str->cap + num_bytes;
	new_str->str = reallocate(str->str, str->cap, new_str->cap);
	memmove(&new_str->str[num_bytes], new_str->str, str->cap);
	memcpy(new_str->str, temp, num_bytes);

	str->cap = 0;
	str->str = NULL;
	return new_str;
}

ddpstring* _ddp_string_char_verkettet(ddpstring* str, ddpchar c) {
	ddpstring* new_str = ALLOCATE(ddpstring, 1);
	DBGLOG("_ddp_string_char_verkettet: %p", new_str);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

	new_str->cap = str->cap + num_bytes;
	new_str->str = reallocate(str->str, str->cap, new_str->cap);
	memcpy(&new_str->str[str->cap - 1], temp, num_bytes);
	new_str->str[new_str->cap - 1] = '\0';

	str->str = NULL;
	str->cap = 0;
	return new_str;
}

ddpint _ddp_string_to_int(ddpstring* str) {
	if (str->cap == 0 || str->str[0] == '\0') return 0; // empty string

	return strtoll(str->str, NULL, 10); // cast the copy to int
}

ddpfloat _ddp_string_to_float(ddpstring* str) {
	if (str->cap == 0 || str->str[0] == '\0') return 0; // empty string

	return strtod(str->str, NULL); // maybe works with comma seperator? We'll see
}

ddpstring* _ddp_int_to_string(ddpint i) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("_ddp_int_to_string: %p", dstr);

	char buffer[21];
	int len = sprintf(buffer, "%lld", i);

	char* string = ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	dstr->str = string;
	dstr->cap = len + 1;
	return dstr;
}

ddpstring* _ddp_float_to_string(ddpfloat f) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("_ddp_float_to_string: %p", dstr);

	char buffer[50];
	int len = sprintf(buffer, "%.16g", f);

	char* string = ALLOCATE(char, len + 1); // the char array of the string + null-terminator
	memcpy(string, buffer, len);
	string[len] = '\0';

	// set the string fields
	dstr->str = string;
	dstr->cap = len + 1;
	return dstr;
}

ddpstring* _ddp_bool_to_string(ddpbool b) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("_ddp_bool_to_string: %p", dstr);

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

ddpstring* _ddp_char_to_string(ddpchar c) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("_ddp_bool_to_string: %p", dstr);

	char temp[5];
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // invalid utf8, string will be empty
		num_bytes = 0;
	}

	char* string = ALLOCATE(char, num_bytes + 1);
	memcpy(string, temp, num_bytes);
	string[num_bytes] = '\0';

	dstr->cap = num_bytes + 1;
	dstr->str = string;
	return dstr;
}

ddpbool _ddp_string_equal(ddpstring* str1, ddpstring* str2) {
	if (str1 == str2) return true;
	if (strlen(str1->str) != strlen(str2->str)) return false; // if the length is different, it's a quick false return
	return memcmp(str1->str, str2->str, str1->cap) == 0;
}