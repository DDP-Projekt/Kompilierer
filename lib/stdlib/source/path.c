#include "ddpmemory.h"
#include "ddptypes.h"
#include "winapi-path.h"
#include <stdio.h>
#include <string.h>


void Windows_Saeubern(ddpstring *ret, ddpstring *path) {
	char cleaned[MAX_PATH];
	if (!PathCanonicalize(cleaned, path->str)) {
		*ret = DDP_EMPTY_STRING;
		ret->str = DDP_ALLOCATE(char, 1);
		ret->str[0] = '\0';
		ret->cap = 1;
		return;
	}
	int len = strlen(cleaned) + 1;
	char *str = DDP_ALLOCATE(char, len);
	memcpy(str, cleaned, len);
	ret->str = str;
	ret->cap = len;
}

void Windows_Pfad_Verbinden(ddpstring *ret, ddpstring *a, ddpstring *b) {
	char joined[MAX_PATH];
	if (!PathCombine(joined, a->str, b->str)) {
		*ret = DDP_EMPTY_STRING;
		ret->str = DDP_ALLOCATE(char, 1);
		ret->str[0] = '\0';
		ret->cap = 1;
		return;
	}

	int len = strlen(joined) + 1;
	char *str = DDP_ALLOCATE(char, len);
	memcpy(str, joined, len);
	ret->str = str;
	ret->cap = len;
}
