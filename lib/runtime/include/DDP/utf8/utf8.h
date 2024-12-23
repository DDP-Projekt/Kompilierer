/*
    copied and modified from https://gist.github.com/stanislaw/f62c36823242c4ffea1b
    further information on utf8 bit info can be found here https://stackoverflow.com/questions/5012803/test-if-char-string-contains-multibyte-characters
*/
#ifndef DDP_UTF8_H
#define DDP_UTF8_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>


bool utf8_is_continuation(char c);

// checks if this byte is part of a multibyte sequence
bool utf8_is_multibyte(char c);

// returns the number of bytes of the Unicode char which c is the first byte of
// 0 indicates error
int utf8_indicated_num_bytes(char c);

// returns the number of unicode characters in s
// s must be null-terminated
size_t utf8_strlen(char *s);

// check if the first unicode character in c is a single-byte character
bool utf8_is_single_byte(char *c);
// check if the first unicode character in c is a double-byte character
bool utf8_is_double_byte(char *c);
// check if the first unicode character in c is a triple-byte character
bool utf8_is_triple_byte(char *c);
// check if the first unicode character in c is a quadruple-byte character
bool utf8_is_quadruple_byte(char *c);

// returns the number of bytes of the first unicode character in s
// s must be null-terminated
size_t utf8_num_bytes(char *s);

// returns the number of bytes needed to encode this character in utf8
size_t utf8_num_bytes_char(uint32_t c);

// decodes the unicode character c into s
// s must be at least 5 chars long and will be null-terminated by the functions
// returns the number of bytes in c
// returns -1 if c is not a valid utf8 character
size_t utf8_char_to_string(char *s, int32_t c);

// decode the first codepoint in str into out
// str must be null-terminated
// returns the number of bytes encoded into out or -1 if str was invalid utf8
size_t utf8_string_to_char(char *str, uint32_t *out);

#endif // DDP_UTF8_H
