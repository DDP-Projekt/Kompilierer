/*
    copied and modified from https://gist.github.com/stanislaw/f62c36823242c4ffea1b
*/
#ifndef DDP_UTF8_H
#define DDP_UTF8_H

#include <stdint.h>
#include <stdbool.h>
#include <stddef.h>

bool utf8_is_continuation(char c);

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

// returns the number of bytes of the first unicode character in s
// s must be null-terminated
size_t utf8_num_bytes(char* s);

// decodes the unicode character c into s
// s must be at least 5 chars long and will be null-terminated by the functions
// returns the number of bytes in c (the resulting strlen of s)
int utf8_char_to_string(char* s, int32_t c);

#endif // DDP_UTF8_H