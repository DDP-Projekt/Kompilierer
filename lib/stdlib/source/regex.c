#include "ddptypes.h"
#define PCRE2_STATIC
#define PCRE2_CODE_UNIT_WIDTH 8
#include <pcre2.h>
#include "error.h"
#include <string.h>
#include <stdio.h>
#include <math.h>
#include <ddpmemory.h>

typedef struct Treffer {
	ddpstring text;
	ddpstringlist groups;
} Treffer;

typedef struct TrefferList {
	Treffer* arr;
	ddpint len;
	ddpint cap;
} TrefferList;

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
		ddp_error("Regex-Feher in '%s' bei Spalte %d: %s\n", false, muster->str, (int)erroroffset, buffer);
	}

	return re;
}

void make_Treffer(Treffer *tr, pcre2_match_data *match_data, int capture_count){
	PCRE2_UCHAR *substring;
	PCRE2_SIZE substring_length;
	pcre2_substring_get_bynumber(match_data, 0, &substring, &substring_length);

	ddp_string_from_constant(&tr->text, (char*)substring);
	ddp_ddpstringlist_from_constants(&tr->groups, capture_count-1);
	
	for (int i = 1; i < capture_count; i++) {
		pcre2_substring_get_bynumber(match_data, i, &substring, &substring_length);
		
		ddp_string_from_constant(&tr->groups.arr[i-1], (char*)substring);
	}
	pcre2_substring_free(substring);
}

void Regex_Erster_Treffer(Treffer *ret, ddpstring *muster, ddpstring *text) {
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
			ddp_string_from_constant(&ret->text, "");
			ddp_ddpstringlist_from_constants(&ret->groups, 0);
		}
		else {
			PCRE2_UCHAR buffer[256];
			pcre2_get_error_message(rc, buffer, sizeof(buffer));
			ddp_error("Match-Fehler in '%s': %s\n", true, muster->str, buffer);
		}
	}
	else {
		make_Treffer(ret, match_data, rc);
	}

	// Free up the regular expression and match data
	pcre2_code_free(re);
	pcre2_match_data_free(match_data);
}

void Regex_N_Treffer(TrefferList *ret, ddpstring *muster, ddpstring *text, ddpint n) {
	PCRE2_SPTR pattern = (PCRE2_SPTR)muster->str; // The regex pattern
	PCRE2_SPTR subject = (PCRE2_SPTR)text->str; // The string to match against

	pcre2_code *re = compile_regex(pattern, subject, muster);
	if (re == NULL) return;

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);

	// Initialize an empty list into ret
	ret->len = 0;
	ret->cap = 0;
	ret->arr = ALLOCATE(Treffer, 0);

	PCRE2_SIZE start_offset = 0;
	int i = 0;
	// Perform the match
	while(i < n || n == -1) {
		int rc = pcre2_match(
			re,			// the compiled pattern
			subject,	// the subject string
			text->cap,	// the length of the subject
			start_offset,
			0,			// default options
			match_data,	// block for storing the result
			NULL		// use default match context
		);

		if (rc < 0) {
			if (rc != PCRE2_ERROR_NOMATCH) {
				PCRE2_UCHAR buffer[256];
				pcre2_get_error_message(rc, buffer, sizeof(buffer));
				ddp_error("Match-Fehler in '%s': %s\n", false, muster->str, buffer);
			}
			break;
		}

		Treffer tr;
		make_Treffer(&tr, match_data, rc);
		
		// incrase list size if needed
		if (ret->len == ret->cap) {
			ddpint old_cap = ret->cap;
			ret->cap = GROW_CAPACITY(ret->cap);
			ret->arr = ddp_reallocate(ret->arr, old_cap * sizeof(Treffer), ret->cap * sizeof(Treffer));
		}

		// append new element
		memcpy(&((uint8_t*)ret->arr)[ret->len * sizeof(Treffer)], &tr, sizeof(Treffer));
		ret->len++;

		start_offset = pcre2_get_ovector_pointer(match_data)[1];
		i++;
	}

	// Free up the regular expression and match data
	pcre2_code_free(re);
	pcre2_match_data_free(match_data);
}

// TODO: subtitute

// return true if regex is a valid pcre regex
bool Ist_Regex(ddpstring *muster) {
	int errornumber;
	size_t erroroffset;

	return pcre2_compile((PCRE2_SPTR)muster->str, PCRE2_ZERO_TERMINATED, 0, &errornumber, &erroroffset, NULL) != NULL;
}