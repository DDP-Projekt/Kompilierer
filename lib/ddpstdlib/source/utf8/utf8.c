/*
    copied and modified from https://gist.github.com/stanislaw/f62c36823242c4ffea1b
*/
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "utf8/utf8.h"

// validate that s is valid utf8
// s must be null-terminated
bool utf8_is_valid(char* s) {
    int32_t i = 0;
    size_t len = strlen(s);
    
    while (i < len) {        
        size_t num_bytes = utf8_num_bytes(s + i);
        
        if (num_bytes != 0) {
            i += num_bytes;
        } else {
            return false;
        }
    }
    
    return true;
}

// check if the first unicode character in c is a single-byte character
bool utf8_is_single_byte(char* c) {
    return (c[0] & 0x80) == 0x0;
}

// check if the first unicode character in c is a double-byte character
bool utf8_is_double_byte(char* c) {
    return (c[0] & 0xe0) == 0xc0 && utf8_is_continuation(c[1]);
}

// check if the first unicode character in c is a triple-byte character
bool utf8_is_triple_byte(char* c) {
    return (c[0] & 0xf0) == 0xe0 && utf8_is_continuation(c[1]) && utf8_is_continuation(c[2]);
}

// check if the first unicode character in c is a quadruple-byte character
bool utf8_is_quadruple_byte(char* c) {
    return (c[0] & 0xf8) == 0xf0 && utf8_is_continuation(c[1]) && utf8_is_continuation(c[2]) && utf8_is_continuation(c[3]);
}

bool utf8_is_continuation(char c) {
    return (c & 0xc0) == 0x80;
}

// returns the number of unicode characters in s
// s must be null-terminated
size_t utf8_strlen(char* s) {
    size_t i = 0, len = 0;
    while(s[i] != 0) { // while not at null-terminator
        if ( ! utf8_is_continuation(s[i])) ++len;
        ++i;
    }
    return len;
}

// returns the number of bytes of the first unicode character in s
// s must be null-terminated
size_t utf8_num_bytes(char* s) {
    size_t len = strlen(s), num_bytes = 0;
    
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

// returns a new string with the unicode char at n removed
// s must be null-terminated
char* utf8_remove_char(char* s, size_t n) {
    size_t len = strlen(s);
    if (len < n) { // index out of range
        return NULL;
    }
    
    size_t num_shifts = utf8_num_bytes(s + n); // how many bytes will be removed
    char* new_string = malloc(len * sizeof(char));
    
    memcpy(new_string, s, n); // copy up to the removed char
    memcpy(new_string + n, s + n + num_shifts, len - n - num_shifts + 1); // copy everything after the removed char
    
    return new_string;
}

// returns a new string with c added at n
// both s and c must be null-terminated
char* utf8_add_char(char* s, char* c, size_t n) {
    size_t len = strlen(s);
    if (len < n) {
        return NULL;
    }
    
    size_t num_shifts = utf8_num_bytes(c);
    char* new_string = malloc((len + num_shifts + 1) * sizeof(char));
    
    // copy the begining of the string
    memcpy(new_string, s, n);
    
    // add the new char
    memcpy(new_string + n, c, num_shifts);
    
    // copy the remaining characters
    memcpy(new_string + n + num_shifts, s + n, len - n + 1);
    
    return new_string;
}

// replace the first occurence of needle in haystack with replace
// all parameters must be null-terminated
char* utf8_replace(char* needle, char* replace, char* haystack) {
    size_t
        len_replace = strlen(replace),
        len_needle = strlen(needle),
        len = strlen(haystack);
    
    int32_t diff = (int32_t) (len_replace - len_needle); // size difference
    
    char* new_string = calloc((len + diff + 1), sizeof(char)); // new zeroed string
    
    char* pos = strstr(haystack, needle); // find the needle
    
    if (pos == NULL) { // needle not found, return a copy
        strcpy(new_string, haystack);
        return new_string;
    }
    
    size_t num_shifts = pos - haystack;
    
    // Add begining of the string
    memcpy(new_string, haystack, num_shifts);
    
    // Copy the replacement in place of the needle
    memcpy(new_string + num_shifts, replace, len_replace);
    
    // Copy the remainder of the initial string
    memcpy(new_string + num_shifts + len_replace, pos + len_needle, len - num_shifts - len_needle);
    
    return new_string;
}

// replace all occurences of needle in haystack with replace
// all parameters must be null-terminated
char* utf8_replace_all(char* needle, char* replace, char* haystack) {
    char
        * new_string = utf8_replace(needle, replace, haystack),
        * old_new_string = NULL;
    
    while (strstr(new_string, needle) != NULL) {
        old_new_string = new_string;
        new_string = utf8_replace(needle, replace, new_string);
        free(old_new_string);
    }
    
    return new_string;
}