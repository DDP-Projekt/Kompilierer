/*
    copied and modified from https://gist.github.com/stanislaw/f62c36823242c4ffea1b
	further information on utf8 bit info can be found here https://stackoverflow.com/questions/5012803/test-if-char-string-contains-multibyte-characters
*/
#include "DDP/utf8/utf8.h"
#include <string.h>
#include <uchar.h>

// check if the first unicode character in c is a single-byte character
bool utf8_is_single_byte(const char *c) {
	return (c[0] & 0x80) == 0x0;
}

// check if the first unicode character in c is a double-byte character
bool utf8_is_double_byte(const char *c) {
	return (c[0] & 0xe0) == 0xc0 && utf8_is_continuation(c[1]);
}

// check if the first unicode character in c is a triple-byte character
bool utf8_is_triple_byte(const char *c) {
	return (c[0] & 0xf0) == 0xe0 && utf8_is_continuation(c[1]) && utf8_is_continuation(c[2]);
}

// check if the first unicode character in c is a quadruple-byte character
bool utf8_is_quadruple_byte(const char *c) {
	return (c[0] & 0xf8) == 0xf0 && utf8_is_continuation(c[1]) && utf8_is_continuation(c[2]) && utf8_is_continuation(c[3]);
}

bool utf8_is_continuation(char c) {
	return (c & 0xc0) == 0x80;
}

// checks if this byte is part of a multibyte sequence
bool utf8_is_multibyte(char c) {
	return (c & 0x80) == 0x80;
}

// returns the number of bytes of the Unicode char which c is the first byte of
// 0 indicates error
int utf8_indicated_num_bytes(char c) {
	if ((c & 0x80) == 0x0) {
		return 1;
	}
	if ((c & 0xf0) == 0xf0) {
		return 4;
	}
	if ((c & 0xe0) == 0xe0) {
		return 3;
	}
	if ((c & 0xc0) == 0xc0) {
		return 2;
	}
	return 0;
}

// returns the number of unicode characters in s
// s must be null-terminated
size_t utf8_strlen(const char *s) {
	if (s == NULL) {
		return 0;
	}

	size_t i = 0, len = 0;
	while (s[i] != 0) { // while not at null-terminator
		if (!utf8_is_continuation(s[i])) {
			++len;
		}
		++i;
	}
	return len;
}

// returns the number of bytes of the first unicode character in s
// s must be null-terminated
size_t utf8_num_bytes(const char *s) {
	if (s == NULL) {
		return 0;
	}

	// strlen but for maximally 4 characters
	// yields a massive performance gain for big strings (as the O(n) complexity is dropped)
	uint8_t len = 0;
	const char *it = s;
	while (*(it++) != '\0' && len < 4) {
		len++;
	}

	size_t num_bytes = 0;

	// is valid single byte (ie 0xxx xxxx)
	if (len >= 1 && utf8_is_single_byte(s)) {
		num_bytes = 1;

		// or is valid double byte (ie 110x xxxx and continuation byte)
	} else if (len >= 2 && utf8_is_double_byte(s)) {
		num_bytes = 2;

		// or is valid tripple byte (ie 1110 xxxx and continuation byte)
	} else if (len >= 3 && utf8_is_triple_byte(s)) {
		num_bytes = 3;

		// or is valid tripple byte (ie 1111 0xxx and continuation byte)
	} else if (len >= 4 && utf8_is_quadruple_byte(s)) {
		num_bytes = 4;
	}

	return num_bytes;
}

// returns the number of bytes needed to encode this character in utf8
size_t utf8_num_bytes_char(uint32_t c) {
	if ((int32_t)c < 0) {
		return -1;
	} else if (c <= ((1 << 7) - 1)) {
		return 1;
	} else if (c <= ((1 << 11) - 1)) {
		return 2;
	} else if (0xD800 <= c && c <= 0xDFFF) {
		return -1;
	} else if (c <= ((1 << 16) - 1)) {
		return 3;
	} else if (c <= 0x0010FFFF) {
		return 4;
	}
	return -1;
}

static mbstate_t state;

// decodes the unicode character c into s
// s must be at least 5 chars long and will be null-terminated by the functions
// returns the number of bytes in c
// returns -1 if c is not a valid utf8 character
size_t utf8_char_to_string(char *s, int32_t c) {
	size_t num_bytes = c32rtomb(s, c, &state);
	if (num_bytes != (size_t)-1) {
		s[num_bytes] = '\0';
	}
	return num_bytes;
}

// decode the first codepoint in str into out
// str must be null-terminated
// returns the number of bytes encoded into out or -1 if str was invalid utf8
size_t utf8_string_to_char(const char *str, uint32_t *out) {
	if (str == NULL) {
		return -1;
	}

	size_t n = utf8_num_bytes(str);
	mbrtoc32(out, str, n, &state);
	return n;
}
