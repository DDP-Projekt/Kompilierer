/*
	defines functions for ddp operators
*/
#include "ddptypes.h"
#include "memory.h"
#include "debug.h"
#include <stdlib.h>
#include <math.h>
#include <float.h>

ddpint inbuilt_int_betrag(ddpint i) {
	return llabs(i);
}

ddpfloat inbuilt_float_betrag(ddpfloat f) {
	return fabs(f);
}

// helper for inbuilt_string_to_int and inbuilt_string_to_float
static bool isNum(ddpchar c) {
	return c >= '0' && c <= '9';
}

// way too huge, must be made better later
ddpint inbuilt_string_to_int(ddpstring* str) {
	if (str->len == 0) return 0;
	ddpint i = 0; // index to copy str
	bool negative = false; // wether or not the number will be negative
	if (str->str[0] == '-') {
		i = 1; // we ignore the sign when copying
		negative = true; // and apply it later outselves
	}
	// strtoll wants a char* not ddpchar* so we copy str->str
	int charsLen = 19; // len of the copy (19 = max base 10 length of 64bit int)
	char* chars = ALLOCATE(char, charsLen); // allocate the copy
	// copy str until we encounter a non-number
	for (; i < str->len && isNum(str->str[i]); i++) {
		if (i > charsLen) { // number might be larger then 19 letters
			// grow the array if neccessery
			chars = GROW_ARRAY(char, chars, charsLen, GROW_CAPACITY(charsLen));
			charsLen = GROW_CAPACITY(charsLen);
		}
		chars[negative ? i-1 : i] = str->str[i];
	}
	ddpint result = strtoll(chars, NULL, 10); // cast the copy to int
	FREE_ARRAY(char, chars, charsLen); // free the copy
	return negative ? -result : result; // return the result and consider the sign
}

// way too huge, must be made better later
ddpfloat inbuilt_string_to_float(ddpstring* str) {
	if (str->len == 0) return 0;
	ddpint i = 0; // index to copy str
	bool negative = false; // wether or not the number will be negative
	if (str->str[0] == '-') {
		i = 1; // we ignore the sign when copying
		negative = true; // and apply it later outselves
	}
	// strtoll wants a char* not ddpchar* so we copy str->str
	int charsLen = 19; // len of the copy (19 = max base 10 length of 64bit int)
	char* chars = ALLOCATE(char, charsLen); // allocate the copy
	// copy str until we encounter a non-number
	for (; i < str->len && (isNum(str->str[i]) || str->str[i] == ','); i++) {
		if (i > charsLen) { // number might be larger then 19 letters
			// grow the array if neccessery
			chars = GROW_ARRAY(char, chars, charsLen, GROW_CAPACITY(charsLen));
			charsLen = GROW_CAPACITY(charsLen);
		}
		chars[negative ? i-1 : i] = str->str[i];
	}
	DBGLOG("%.*s", charsLen, chars);
	ddpfloat result = strtod(chars, NULL); // cast the copy to int
	FREE_ARRAY(char, chars, charsLen); // free the copy
	return negative ? -result : result; // return the result and consider the sign
}

ddpstring* inbuilt_int_to_string(ddpint i) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_int_to_string: %p", dstr);

	char buffer[10]; // is this buffer large enough?
	int len = sprintf(buffer, "%ld", i);

	ddpchar* string = ALLOCATE(ddpchar, len); // the char array of the string
	// copy the passed char array
	for (int i = 0; i < len; i++) {
		string[i] = (ddpchar)buffer[i];
	}
	// set the string fields
	dstr->str = string;
	dstr->len = len;
	return dstr;
}

ddpstring* inbuilt_float_to_string(ddpfloat f) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_float_to_string: %p", dstr);

	char buffer[50];
	int len = sprintf(buffer, "%g", f);

	ddpchar* string = ALLOCATE(ddpchar, len); // the char array of the string
	// copy the passed char array
	for (int i = 0; i < len; i++) {
		string[i] = (ddpchar)buffer[i];
	}
	// set the string fields
	dstr->str = string;
	dstr->len = len;
	return dstr;
}

ddpstring* inbuilt_bool_to_string(ddpbool b) {
	ddpstring* dstr = ALLOCATE(ddpstring, 1); // up here to log the adress in debug mode
	DBGLOG("inbuilt_bool_to_string: %p", dstr);

	ddpchar* string;

	if (b) {
		dstr->len = 4;
		string = ALLOCATE(ddpchar, 4);
		string[0] = 'w'; string[1] = 'a'; string[2] = 'h'; string[3] = 'r'; 
	} else {
		dstr->len = 6;
		string = ALLOCATE(ddpchar, 6);
		string[0] = 'f'; string[1] = 'a'; string[2] = 'l'; string[3] = 's'; string[4] = 'c'; string[5] = 'h';
	}
	dstr->str = string;
	return dstr;
}

ddpbool inbuilt_string_equal(ddpstring* str1, ddpstring* str2) {
	if (str1->len != str2->len) return false; // if the length is different, it's a quick false return

	// compare each character
	for (ddpint i = 0; i < str1->len; i++) {
		if (str1->str[i] != str2->str[i]) return false;
	}
	
	return true;
}