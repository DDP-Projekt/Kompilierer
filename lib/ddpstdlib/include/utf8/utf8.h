/*
    copied and modified from https://gist.github.com/stanislaw/f62c36823242c4ffea1b
*/
#ifndef DDP_UTF8_H
#define DDP_UTF8_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

bool utf8_is_continuation(char c);

// validate that s is valid utf8
// s must be null-terminated
bool utf8_is_valid(char* s);

// returns the number of unicode characters in s
// s must be null-terminated
size_t utf8_strlen(char* s);

// check if the first unicode character in c is a single-byte character
bool utf8_is_single_byte(char* c);
// check if the first unicode character in c is a double-byte character
bool utf8_is_double_byte(char* c);
// check if the first unicode character in c is a triple-byte character
bool utf8_is_triple_byte(char* c);
// check if the first unicode character in c is a quadruple-byte character
bool utf8_is_quadruple_byte(char* c);

// returns a new string with the unicode char at n removed
// s must be null-terminated
char* utf8_remove_char(char* s, size_t n);

// returns a new string with c added at n
// both s and c must be null-terminated
char* utf8_add_char(char* s, char* c, size_t n);

// replace the first occurence of needle in haystack with replace
// all parameters must be null-terminated
char* utf8_replace(char* needle, char* replace, char* haystack);

// replace all occurences of needle in haystack with replace
// all parameters must be null-terminated
char* utf8_replace_all(char* needle, char* replace, char* haystack);

// returns the number of bytes of the first unicode character in s
// s must be null-terminated
size_t utf8_num_bytes(char* s);

#endif // DDP_UTF8_H