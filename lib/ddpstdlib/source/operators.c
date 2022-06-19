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

ddpint inbuilt_int_betrag(ddpint i) {
	return llabs(i);
}

ddpfloat inbuilt_float_betrag(ddpfloat f) {
	return fabs(f);
}

ddpfloat inbuilt_hoch(ddpfloat f1, ddpfloat f2) {
	return pow(f1, f2);
}

ddpfloat inbuilt_log(ddpfloat numerus, ddpfloat base) {
	return log10(numerus) / log10(base);
}

ddpfloat inbuilt_sin(ddpfloat f) {
	return sin(f);
}

ddpfloat inbuilt_cos(ddpfloat f) {
	return cos(f);
}

ddpfloat inbuilt_tan(ddpfloat f) {
	return tan(f);
}

ddpfloat inbuilt_asin(ddpfloat f) {
	return asin(f);
}

ddpfloat inbuilt_acos(ddpfloat f) {
	return acos(f);
}

ddpfloat inbuilt_atan(ddpfloat f) {
	return atan(f);
}

ddpfloat inbuilt_sinh(ddpfloat f) {
	return sinh(f);	
}

ddpfloat inbuilt_cosh(ddpfloat f) {
	return cosh(f);
}

ddpfloat inbuilt_tanh(ddpfloat f) {
	return tanh(f);
}

ddpint inbuilt_string_length(ddpstring* str) {
	return (ddpint)utf8_strlen(str->str);
}

ddpchar inbuilt_string_index(ddpstring* str, ddpint index) {
	if (index > str->cap || index < 1 || str->cap <= 1) {
		printf("Index außerhalb der Text Länge (Index war %ld, Text Länge war %ld)", index, utf8_strlen(str->str));
		exit(1);
	}
	
	size_t i = 0;
	while (str->str[i] != 0 && index > 1) {
		i += utf8_num_bytes(str->str + i);
		index--;
	}

	if (str->str[i] == 0) {
		printf("Index außerhalb der Text Länge (Index war %ld, Text Länge war %ld)", index, utf8_strlen(str->str));
		exit(1);
	}

	return utf8_string_to_char(str->str + i);
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
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}
	
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
	int num_bytes = utf8_char_to_string(temp, c);
	if (num_bytes == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes = 0;
	}

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
	int num_bytes1 = utf8_char_to_string(temp1, c1);
	if (num_bytes1 == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes1 = 0;
	}

	char temp2[5];
	int num_bytes2 = utf8_char_to_string(temp2, c2);
	if (num_bytes2 == -1) { // if c is invalid utf8, we return simply a copy of str
		num_bytes2 = 0;
	}

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

ddpstring* inbuilt_char_to_string(ddpchar c) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_bool_to_string: %p", dstr);

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

ddpbool inbuilt_string_equal(ddpstring* str1, ddpstring* str2) {
	if (strlen(str1->str) != strlen(str2->str)) return false; // if the length is different, it's a quick false return
	return memcmp(str1->str, str2->str, str1->cap) == 0;
}