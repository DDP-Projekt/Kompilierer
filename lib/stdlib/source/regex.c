#include "ddptypes.h"
#define PCRE2_STATIC
#define PCRE2_CODE_UNIT_WIDTH 8
#include <pcre2.h>
#include "error.h"
#include <string.h>
#include <stdio.h>
#include <math.h>
#include <ddpmemory.h>

pcre2_code* compile_regex(PCRE2_SPTR pattern, PCRE2_SPTR subject, ddpstring *muster) {
    int errornumber;
    size_t erroroffset;

    // Compile the regular expression
    pcre2_code *re = pcre2_compile(
        pattern,				// the pattern
        PCRE2_ZERO_TERMINATED,	// indicates pattern is zero-terminated
        0,						// default options
        &errornumber,			// for error number
        &erroroffset,			// for error offset
        NULL					// use default compile context
	);                 

    // Handle compilation errors
    if (re == NULL) {
        PCRE2_UCHAR buffer[256];
        pcre2_get_error_message(errornumber, buffer, sizeof(buffer));
		ddp_error("Regex-Feher in '%s' bei Spalte %d: %s\n", true, muster->str, (int)erroroffset, buffer);
    }

	return re;
}

void Regex_Erster_Treffer(ddpstring *ret, ddpstring *muster, ddpstring *text) {
	PCRE2_SPTR pattern = (PCRE2_SPTR)muster->str; // The regex pattern
    PCRE2_SPTR subject = (PCRE2_SPTR)text->str; // The string to match against

	pcre2_code *re = compile_regex(pattern, subject, muster);
	if (re == NULL) return;

    // Create the match data
    pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);

    // Perform the match
    int rc = pcre2_match(
        re,			// the compiled pattern
        subject,	// the subject string
        text->cap,	// the length of the subject
        0,			// start at offset 0 in the subject
        0,			// default options
        match_data,	// block for storing the result
        NULL		// use default match context
	);                 

    // Check the result
    if (rc < 0) { // Match failed
		if (rc == PCRE2_ERROR_NOMATCH) {
			ddp_string_from_constant(ret, "");
		}
		else {
			PCRE2_UCHAR buffer[256];
        	pcre2_get_error_message(rc, buffer, sizeof(buffer));
			ddp_error("Match-Fehler in '%s': %s\n", true, muster->str, buffer);
		}
    }
	else {
		PCRE2_SIZE *ovector = pcre2_get_ovector_pointer(match_data);
		int len = ovector[1] - ovector[0];
		char *temp = ALLOCATE(char, len+1);
		strncpy(temp, text->str + ovector[0], len);
		temp[len] = '\0';
		ddp_string_from_constant(ret, temp);
		
		FREE_ARRAY(char, temp, len+1);
    }

    // Free up the regular expression and match data
    pcre2_code_free(re);
    pcre2_match_data_free(match_data);
}

void Regex_Alle_Treffer(ddpstringlist *ret, ddpstring *muster, ddpstring *text) {
    PCRE2_SPTR pattern = (PCRE2_SPTR)muster->str; // The regex pattern
    PCRE2_SPTR subject = (PCRE2_SPTR)text->str; // The string to match against

    pcre2_code *re = compile_regex(pattern, subject, muster);
	if (re == NULL) return;

    // Create the match data
    pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);

	PCRE2_SIZE start_offset = 0;

	ddp_ddpstringlist_from_constants(ret, 0);

    // Perform the match
    while(pcre2_match(
        re,			// the compiled pattern
        subject,	// the subject string
        text->cap,	// the length of the subject
        start_offset,
        0,			// default options
        match_data,	// block for storing the result
        NULL		// use default match context
	) > 0) {
		PCRE2_SIZE *ovector = pcre2_get_ovector_pointer(match_data);
		start_offset = ovector[1];

		int len = ovector[1] - ovector[0];
		char *temp = ALLOCATE(char, len+1);
		strncpy(temp, text->str + ovector[0], len);
		temp[len] = '\0';

		ddpstring ddpstr;
		ddp_string_from_constant(&ddpstr, temp);
		
		if (ret->len == ret->cap) {
			ddpint old_cap = ret->cap;
			ret->cap = GROW_CAPACITY(ret->cap);
			ret->arr = ddp_reallocate(ret->arr, old_cap * sizeof(ddpstring), ret->cap * sizeof(ddpstring));
		}

		memcpy(&((uint8_t*)ret->arr)[ret->len * sizeof(ddpstring)], &ddpstr, sizeof(ddpstring));
		ret->len++;

		FREE_ARRAY(char, temp, len+1);
	}                 

    // Free up the regular expression and match data
    pcre2_code_free(re);
    pcre2_match_data_free(match_data);
}

// return true if regex is a valid pcre regex
bool Ist_Regex(ddpstring *muster) {
	int errornumber;
    size_t erroroffset;

	return pcre2_compile((PCRE2_SPTR)muster->str, PCRE2_ZERO_TERMINATED, 0, &errornumber, &erroroffset, NULL) != NULL;
}