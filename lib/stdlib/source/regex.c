#include "ddptypes.h"
#define PCRE2_STATIC
#define PCRE2_CODE_UNIT_WIDTH 8
#include "error.h"
#include <ddpmemory.h>
#include <pcre2.h>
#include <stdio.h>
#include <string.h>

typedef struct Treffer {
	ddpstring text;
	ddpstringlist gruppen;
} Treffer;

typedef struct TrefferList {
	Treffer *arr;
	ddpint len;
	ddpint cap;
} TrefferList;

static pcre2_code *compile_regex(PCRE2_SPTR pattern, PCRE2_SPTR subject, ddpstring *muster) {
	int errornumber;
	size_t erroroffset;

	// Compile the regular expression
	pcre2_code *re = pcre2_compile(
		pattern,			   // the pattern
		PCRE2_ZERO_TERMINATED, // indicates pattern is zero-terminated
		0,					   // default options
		&errornumber,		   // for error number
		&erroroffset,		   // for error offset
		NULL				   // use default compile context
	);

	// Handle compilation errors
	if (re == NULL) {
		PCRE2_UCHAR buffer[256];
		pcre2_get_error_message(errornumber, buffer, sizeof(buffer));
		ddp_error("Regex-Feher in '" DDP_STRING_FMT "' bei Spalte %d: %s\n", false, muster->str, (int)erroroffset, buffer);
	}

	return re;
}

static void make_Treffer(Treffer *tr, pcre2_match_data *match_data, int capture_count) {
	PCRE2_UCHAR *substring;
	PCRE2_SIZE substring_length;

	ddp_ddpstringlist_from_constants(&tr->gruppen, capture_count - 1);
	for (int i = 0; i < capture_count; i++) {
		pcre2_substring_get_bynumber(match_data, i, &substring, &substring_length);

		ddp_string_from_constant(i == 0 ? &tr->text : &tr->gruppen.arr[i - 1], (char *)substring);

		pcre2_substring_free(substring);
	}
}

void Regex_Erster_Treffer(Treffer *ret, ddpstring *muster, ddpstring *text) {
	DDP_MIGHT_ERROR;

	PCRE2_SPTR pattern = (PCRE2_SPTR)muster->str; // The regex pattern
	PCRE2_SPTR subject = (PCRE2_SPTR)text->str;	  // The string to match against

	pcre2_code *re = compile_regex(pattern, subject, muster);
	if (re == NULL) {
		ret->text = DDP_EMPTY_STRING;
		ddp_ddpstringlist_from_constants(&ret->gruppen, 0);
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		pcre2_code_free(re);
		ret->text = DDP_EMPTY_STRING;
		ddp_ddpstringlist_from_constants(&ret->gruppen, 0);
		return;
	}

	// Perform the match
	int rc = pcre2_match(
		re,			// the compiled pattern
		subject,	// the subject string
		text->cap,	// the length of the subject
		0,			// start at offset 0 in the subject
		0,			// default options
		match_data, // block for storing the result
		NULL		// use default match context
	);

	// Check the result
	if (rc < 0) { // Match failed
		if (rc != PCRE2_ERROR_NOMATCH) {
			PCRE2_UCHAR buffer[256];
			pcre2_get_error_message(rc, buffer, sizeof(buffer));
			ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", true, muster->str, buffer);
		}

		ret->text = DDP_EMPTY_STRING;
		ddp_ddpstringlist_from_constants(&ret->gruppen, 0);
	} else {
		make_Treffer(ret, match_data, rc);
	}

	// Free up the regular expression and match data
	pcre2_code_free(re);
	pcre2_match_data_free(match_data);
}

void Regex_N_Treffer(TrefferList *ret, ddpstring *muster, ddpstring *text, ddpint n) {
	DDP_MIGHT_ERROR;

	PCRE2_SPTR pattern = (PCRE2_SPTR)muster->str; // The regex pattern
	PCRE2_SPTR subject = (PCRE2_SPTR)text->str;	  // The string to match against

	// Initialize an empty list into ret
	*ret = DDP_EMPTY_LIST(TrefferList);

	pcre2_code *re = compile_regex(pattern, subject, muster);
	if (re == NULL) {
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		pcre2_code_free(re);
		return;
	}

	PCRE2_SIZE start_offset = 0;
	int i = 0;
	// Perform the match
	while (i < n || n == -1) {
		int rc = pcre2_match(
			re,		   // the compiled pattern
			subject,   // the subject string
			text->cap, // the length of the subject
			start_offset,
			0,			// default options
			match_data, // block for storing the result
			NULL		// use default match context
		);

		if (rc < 0) {
			if (rc != PCRE2_ERROR_NOMATCH) {
				PCRE2_UCHAR buffer[256];
				pcre2_get_error_message(rc, buffer, sizeof(buffer));
				ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", false, muster->str, buffer);
			}
			break;
		}

		Treffer tr;
		make_Treffer(&tr, match_data, rc);

		// incrase list size if needed
		if (ret->len == ret->cap) {
			ddpint old_cap = ret->cap;
			ret->cap = DDP_GROW_CAPACITY(ret->cap);
			ret->arr = ddp_reallocate(ret->arr, old_cap * sizeof(Treffer), ret->cap * sizeof(Treffer));
		}

		// append new element
		memcpy(&((uint8_t *)ret->arr)[ret->len * sizeof(Treffer)], &tr, sizeof(Treffer));
		ret->len++;

		start_offset = pcre2_get_ovector_pointer(match_data)[1];
		i++;
	}

	// Free up the regular expression and match data
	pcre2_code_free(re);
	pcre2_match_data_free(match_data);
}

#define SUBSTITUTE_BUFFER_SIZE 2048
static void substitute(ddpstring *ret, ddpstring *muster, ddpstring *text, ddpstring *ersatz, bool all) {
	PCRE2_SPTR pattern = (PCRE2_SPTR)muster->str;	  // The regex pattern
	PCRE2_SPTR subject = (PCRE2_SPTR)text->str;		  // The string to match against
	PCRE2_SPTR replacement = (PCRE2_SPTR)ersatz->str; // The replacement string

	*ret = DDP_EMPTY_STRING;
	pcre2_code *re = compile_regex(pattern, subject, muster);
	if (re == NULL) {
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		pcre2_code_free(re);
		return;
	}

	PCRE2_UCHAR result[SUBSTITUTE_BUFFER_SIZE] = "";
	size_t result_length = SUBSTITUTE_BUFFER_SIZE;
	// Perform the match
	int rc = pcre2_substitute(
		re,		 // the compiled pattern
		subject, // the subject string
		PCRE2_ZERO_TERMINATED,
		0, // start at offset 0 in the subject
		all ? PCRE2_SUBSTITUTE_GLOBAL : 0,
		match_data,	 // block for storing the result
		NULL,		 // use default match context
		replacement, // the replacement string
		PCRE2_ZERO_TERMINATED,
		result,		   // where to put the result
		&result_length // where to put the result length
	);

	// Free up the regular expression and match data
	pcre2_code_free(re);
	pcre2_match_data_free(match_data);

	// Check the result
	if (rc < 0) {
		if (rc != PCRE2_ERROR_NOMATCH) {
			PCRE2_UCHAR buffer[256];
			pcre2_get_error_message(rc, buffer, sizeof(buffer));
			ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", false, muster->str, buffer);
		}
	}

	ddp_string_from_constant(ret, (char *)result);
}

void Regex_Erster_Treffer_Ersetzen(ddpstring *ret, ddpstring *muster, ddpstring *text, ddpstring *ersatz) {
	DDP_MIGHT_ERROR;

	substitute(ret, muster, text, ersatz, false);
}

void Regex_Alle_Treffer_Ersetzen(ddpstring *ret, ddpstring *muster, ddpstring *text, ddpstring *ersatz) {
	DDP_MIGHT_ERROR;

	substitute(ret, muster, text, ersatz, true);
}

void Regex_Spalten(ddpstringlist *ret, ddpstring *muster, ddpstring *text) {
	DDP_MIGHT_ERROR;

	PCRE2_SPTR pattern = (PCRE2_SPTR)muster->str; // The regex pattern
	PCRE2_SPTR subject = (PCRE2_SPTR)text->str;	  // The string to match against

	// Initialize an empty list into ret
	ddp_ddpstringlist_from_constants(ret, 0);

	pcre2_code *re = compile_regex(pattern, subject, muster);
	if (re == NULL) {
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		pcre2_code_free(re);
		return;
	}

	PCRE2_SIZE start_offset = 0;
	// Perform the match
	while (true) {
		int rc = pcre2_match(
			re,		   // the compiled pattern
			subject,   // the subject string
			text->cap, // the length of the subject
			start_offset,
			0,			// default options
			match_data, // block for storing the result
			NULL		// use default match context
		);

		if (rc < 0) {
			if (rc != PCRE2_ERROR_NOMATCH) {
				PCRE2_UCHAR buffer[256];
				pcre2_get_error_message(rc, buffer, sizeof(buffer));
				ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", false, muster->str, buffer);
			}
			break;
		}

		int start = pcre2_get_ovector_pointer(match_data)[0];

		ddpstring r;
		r.str = DDP_ALLOCATE(char, start - start_offset + 1);
		strncpy(r.str, text->str + start_offset, start - start_offset);

		r.str[start - start_offset] = '\0';
		r.cap = start - start_offset + 1;

		// incrase list size if needed
		if (ret->len == ret->cap) {
			ddpint old_cap = ret->cap;
			ret->cap = DDP_GROW_CAPACITY(ret->cap);
			ret->arr = ddp_reallocate(ret->arr, old_cap * sizeof(Treffer), ret->cap * sizeof(Treffer));
		}

		// append new element
		memcpy(&((uint8_t *)ret->arr)[ret->len * sizeof(ddpstring)], &r, sizeof(ddpstring));
		ret->len++;

		start_offset = pcre2_get_ovector_pointer(match_data)[1];
	}

	// incrase list size if needed
	if (ret->len == ret->cap) {
		ddpint old_cap = ret->cap;
		ret->cap = DDP_GROW_CAPACITY(ret->cap);
		ret->arr = ddp_reallocate(ret->arr, old_cap * sizeof(Treffer), ret->cap * sizeof(Treffer));
	}

	ddpstring r;
	r.str = DDP_ALLOCATE(char, strlen(text->str) - start_offset + 1);
	strncpy(r.str, text->str + start_offset, strlen(text->str) - start_offset);

	r.str[strlen(text->str) - start_offset] = '\0';
	r.cap = strlen(text->str) - start_offset + 1;

	// append new element
	memcpy(&((uint8_t *)ret->arr)[ret->len * sizeof(ddpstring)], &r, sizeof(ddpstring));
	ret->len++;

	// Free up the regular expression and match data
	pcre2_code_free(re);
	pcre2_match_data_free(match_data);
}

// return true if regex is a valid pcre regex
ddpbool Ist_Regex(ddpstring *muster) {
	int errornumber;
	size_t erroroffset;

	pcre2_code *re = pcre2_compile((PCRE2_SPTR)muster->str, PCRE2_ZERO_TERMINATED, 0, &errornumber, &erroroffset, NULL);
	if (re == NULL) {
		return false;
	}
	pcre2_code_free(re);
	return true;
}
