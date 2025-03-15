#include "DDP/ddptypes.h"
#define PCRE2_STATIC
#define PCRE2_CODE_UNIT_WIDTH 8
#include "DDP/ddpmemory.h"
#include "DDP/error.h"
#include "DDP/utf8/utf8.h"
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

typedef struct Regex {
	ddpstring ausdruck;
	pcre2_code *obj;
} Regex;

static pcre2_code *compile_regex(PCRE2_SPTR pattern) {
	if (pattern == NULL) {
		ddp_error("Regex-Fehler: Muster war NULL", false);
		return NULL;
	}

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
		ddp_error("Regex-Feher in '" DDP_STRING_FMT "' bei Spalte %d: %s\n", false, pattern, (int)erroroffset, buffer);
	}

	return re;
}

static void make_Treffer(Treffer *tr, pcre2_match_data *match_data, int capture_count) {
	int num_capture_groups = pcre2_get_ovector_count(match_data);

	ddp_ddpstringlist_from_constants(&tr->gruppen, num_capture_groups - 1);
	for (int i = 0; i < num_capture_groups; i++) {
		ddpstring *dest = i == 0 ? &tr->text : &tr->gruppen.arr[i - 1];

		PCRE2_SIZE substr_len;
		int rc;
		switch ((rc = pcre2_substring_length_bynumber(match_data, i, &substr_len))) {
		case 0: // success
			break;
		case PCRE2_ERROR_UNSET:
			*dest = DDP_EMPTY_STRING;
			continue;
		case PCRE2_ERROR_NOSUBSTRING:
		case PCRE2_ERROR_UNAVAILABLE:
			ddp_error("Regex-Fehler: Keine Gruppe mit Nummer %d vorhanden", false, i);
			continue;
		default:
			ddp_error("Regex-Fehler: Die Länge von Gruppe %d konnte nicht bestimmt werden: %d", false, i, rc);
			continue;
		}

		dest->cap = substr_len + 1;
		dest->str = DDP_ALLOCATE(char, dest->cap);
		substr_len = dest->cap;

		switch (pcre2_substring_copy_bynumber(match_data, i, (PCRE2_UCHAR8 *)dest->str, &substr_len)) {
		case PCRE2_ERROR_NOMEMORY:
			ddp_runtime_error(1, "out of memory during regex parsing: %lld", substr_len);
			continue;
		}
	}
}

void regex_first_match(Treffer *ret, pcre2_code *re, char *pattern, char *text) {
	if (re == NULL) {
		ret->text = DDP_EMPTY_STRING;
		ddp_ddpstringlist_from_constants(&ret->gruppen, 0);
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		ret->text = DDP_EMPTY_STRING;
		ddp_ddpstringlist_from_constants(&ret->gruppen, 0);
		return;
	}

	// Perform the match
	int rc = pcre2_match(
		re,				   // the compiled pattern
		(PCRE2_SPTR)text,  // the subject string
		utf8_strlen(text), // the length of the subject
		0,				   // start at offset 0 in the subject
		0,				   // default options
		match_data,		   // block for storing the result
		NULL			   // use default match context
	);

	// Check the result
	if (rc < 0) { // Match failed
		if (rc != PCRE2_ERROR_NOMATCH) {
			PCRE2_UCHAR buffer[256];
			pcre2_get_error_message(rc, buffer, sizeof(buffer));
			ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", true, pattern, buffer);
		}

		ret->text = DDP_EMPTY_STRING;
		ddp_ddpstringlist_from_constants(&ret->gruppen, 0);
	} else {
		make_Treffer(ret, match_data, rc);
	}

	pcre2_match_data_free(match_data);
}

void regex_n_match(TrefferList *ret, pcre2_code *re, char *pattern, char *text, ddpint n) {
	// Initialize an empty list into ret
	*ret = DDP_EMPTY_LIST(TrefferList);

	if (re == NULL) {
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		return;
	}

	PCRE2_SIZE start_offset = 0;
	int i = 0;
	// Perform the match
	while (i < n || n == -1) {
		int rc = pcre2_match(
			re,				   // the compiled pattern
			(PCRE2_SPTR)text,  // the subject string
			utf8_strlen(text), // the length of the subject
			start_offset,
			0,			// default options
			match_data, // block for storing the result
			NULL		// use default match context
		);

		if (rc < 0) {
			if (rc != PCRE2_ERROR_NOMATCH) {
				PCRE2_UCHAR buffer[256];
				pcre2_get_error_message(rc, buffer, sizeof(buffer));
				ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", false, pattern, buffer);
			}
			break;
		}

		Treffer tr;
		make_Treffer(&tr, match_data, rc);

		// incrase list size if needed
		if (ret->len == ret->cap) {
			ddpint old_cap = ret->cap;
			ret->cap = DDP_GROW_CAPACITY(ret->cap);
			ret->arr = DDP_GROW_ARRAY(Treffer, ret->arr, old_cap, ret->cap);
		}

		// append new element
		memcpy(&((uint8_t *)ret->arr)[ret->len * sizeof(Treffer)], &tr, sizeof(Treffer));
		ret->len++;

		start_offset = pcre2_get_ovector_pointer(match_data)[1];
		i++;
	}

	pcre2_match_data_free(match_data);
}

#define SUBSTITUTE_BUFFER_SIZE 2048
static void substitute(ddpstring *ret, pcre2_code *re, char *pattern, char *text, ddpstring *ersatz, bool all) {
	*ret = DDP_EMPTY_STRING;
	if (re == NULL) {
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		return;
	}

	PCRE2_UCHAR result[SUBSTITUTE_BUFFER_SIZE] = "";
	size_t result_length = SUBSTITUTE_BUFFER_SIZE;
	// Perform the match
	int rc = pcre2_substitute(
		re,				  // the compiled pattern
		(PCRE2_SPTR)text, // the subject string
		PCRE2_ZERO_TERMINATED,
		0,								   // start at offset 0 in the subject
		all ? PCRE2_SUBSTITUTE_GLOBAL : 0, // subtitute all
		match_data,						   // block for storing the result
		NULL,							   // use default match context
		(PCRE2_SPTR)ersatz->str,		   // the replacement string
		PCRE2_ZERO_TERMINATED,
		result,		   // where to put the result
		&result_length // where to put the result length
	);

	pcre2_match_data_free(match_data);

	// Check the result
	if (rc < 0) {
		if (rc != PCRE2_ERROR_NOMATCH) {
			PCRE2_UCHAR buffer[256];
			pcre2_get_error_message(rc, buffer, sizeof(buffer));
			ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", false, pattern, buffer);
		}
	}

	ddp_string_from_constant(ret, (char *)result);
}

void regex_split(ddpstringlist *ret, pcre2_code *re, char *pattern, char *text) {
	// Initialize an empty list into ret
	ddp_ddpstringlist_from_constants(ret, 0);

	if (re == NULL) {
		return;
	}

	// Create the match data
	pcre2_match_data *match_data = pcre2_match_data_create_from_pattern(re, NULL);
	if (match_data == NULL) {
		return;
	}

	PCRE2_SIZE start_offset = 0;
	const size_t text_u8_len = utf8_strlen(text);
	// Perform the match
	while (true) {
		int rc = pcre2_match(
			re,				  // the compiled pattern
			(PCRE2_SPTR)text, // the subject string
			text_u8_len,	  // the length of the subject
			start_offset,
			0,			// default options
			match_data, // block for storing the result
			NULL		// use default match context
		);

		if (rc < 0) {
			if (rc != PCRE2_ERROR_NOMATCH) {
				PCRE2_UCHAR buffer[256];
				pcre2_get_error_message(rc, buffer, sizeof(buffer));
				ddp_error("Match-Fehler in '" DDP_STRING_FMT "': %s\n", false, pattern, buffer);
			}
			break;
		}

		int start = pcre2_get_ovector_pointer(match_data)[0];

		ddpstring r;
		r.str = DDP_ALLOCATE(char, start - start_offset + 1);
		strncpy(r.str, text + start_offset, start - start_offset);

		r.str[start - start_offset] = '\0';
		r.cap = start - start_offset + 1;

		// increase list size if needed
		if (ret->len == ret->cap) {
			ddpint old_cap = ret->cap;
			ret->cap = DDP_GROW_CAPACITY(ret->cap);
			ret->arr = DDP_GROW_ARRAY(ddpstring, ret->arr, old_cap, ret->cap);
		}

		// append new element
		ret->arr[ret->len] = r;
		ret->len++;

		start_offset = pcre2_get_ovector_pointer(match_data)[1];
	}

	// increase list size if needed
	if (ret->len == ret->cap) {
		ddpint old_cap = ret->cap;
		ret->cap = DDP_GROW_CAPACITY(ret->cap);
		ret->arr = DDP_GROW_ARRAY(ddpstring, ret->arr, old_cap, ret->cap);
	}

	const ddpint text_len = strlen(text);
	ddpstring r;
	r.str = DDP_ALLOCATE(char, text_len - start_offset + 1);
	strncpy(r.str, text + start_offset, text_len - start_offset);

	r.str[text_len - start_offset] = '\0';
	r.cap = text_len - start_offset + 1;

	// append new element
	ret->arr[ret->len] = r;
	ret->len++;

	pcre2_match_data_free(match_data);
}

// DDP Funktionen

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

void Regex_Erster_Treffer(Treffer *ret, ddpstring *muster, ddpstring *text) {
	DDP_MIGHT_ERROR;

	pcre2_code *re = compile_regex((PCRE2_SPTR)muster->str);
	regex_first_match(ret, re, muster->str, text->str);
	pcre2_code_free(re);
}

void Regex_N_Treffer(TrefferList *ret, ddpstring *muster, ddpstring *text, ddpint n) {
	DDP_MIGHT_ERROR;

	pcre2_code *re = compile_regex((PCRE2_SPTR)muster->str);
	regex_n_match(ret, re, muster->str, text->str, n);
	pcre2_code_free(re);
}

void Regex_Erster_Treffer_Ersetzen(ddpstring *ret, ddpstring *muster, ddpstring *text, ddpstring *ersatz) {
	DDP_MIGHT_ERROR;

	pcre2_code *re = compile_regex((PCRE2_SPTR)muster->str);
	substitute(ret, re, muster->str, text->str, ersatz, false);
	pcre2_code_free(re);
}

void Regex_Alle_Treffer_Ersetzen(ddpstring *ret, ddpstring *muster, ddpstring *text, ddpstring *ersatz) {
	DDP_MIGHT_ERROR;

	pcre2_code *re = compile_regex((PCRE2_SPTR)muster->str);
	substitute(ret, re, muster->str, text->str, ersatz, true);
	pcre2_code_free(re);
}

void Regex_Spalten(ddpstringlist *ret, ddpstring *muster, ddpstring *text) {
	DDP_MIGHT_ERROR;

	pcre2_code *re = compile_regex((PCRE2_SPTR)muster->str);
	regex_split(ret, re, muster->str, text->str);
	pcre2_code_free(re);
}

// Kompiliert

void Regex_Kompilieren(Regex *ret, ddpstring *muster) {
	DDP_MIGHT_ERROR;

	ddp_string_from_constant(&ret->ausdruck, muster->str);
	ret->obj = compile_regex((PCRE2_SPTR)muster->str);
}

void Regex_Kompiliert_Free(Regex *regex) {
	pcre2_code_free(regex->obj);
}

void Regex_Kompiliert_Erster_Treffer(Treffer *ret, Regex *regex, ddpstring *text) {
	DDP_MIGHT_ERROR;
	regex_first_match(ret, regex->obj, regex->ausdruck.str, text->str);
}

void Regex_Kompiliert_N_Treffer(TrefferList *ret, Regex *regex, ddpstring *text, ddpint n) {
	DDP_MIGHT_ERROR;
	regex_n_match(ret, regex->obj, regex->ausdruck.str, text->str, n);
}

void Regex_Kompiliert_Erster_Treffer_Ersetzen(ddpstring *ret, Regex *regex, ddpstring *text, ddpstring *ersatz) {
	DDP_MIGHT_ERROR;
	substitute(ret, regex->obj, regex->ausdruck.str, text->str, ersatz, false);
}

void Regex_Kompiliert_Alle_Treffer_Ersetzen(ddpstring *ret, Regex *regex, ddpstring *text, ddpstring *ersatz) {
	DDP_MIGHT_ERROR;
	substitute(ret, regex->obj, regex->ausdruck.str, text->str, ersatz, true);
}

void Regex_Kompiliert_Spalten(ddpstringlist *ret, Regex *regex, ddpstring *text) {
	DDP_MIGHT_ERROR;
	regex_split(ret, regex->obj, regex->ausdruck.str, text->str);
}
