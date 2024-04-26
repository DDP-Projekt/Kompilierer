#include "ddpmemory.h"
#include "ddptypes.h"
#include "winapi-path.h"
#include <string.h>

// TODO: Use PathCchCanonicalize 
void Windows_Saeubern(ddpstring *ret, ddpstring *path) {
	if (strlen(path->str) + 1 >= DDP_MAX_WIN_PATH) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	char cleaned[DDP_MAX_WIN_PATH];
	if (!PathCanonicalize(cleaned, path->str)) {
		// TODO: Error Handling
		*ret = DDP_EMPTY_STRING;
		return;
	}

	int len = strlen(cleaned) + 1;
	char *str = DDP_ALLOCATE(char, len);
	memcpy(str, cleaned, len);
	ret->str = str;
	ret->cap = len;
}

// TODO: Use PathCchCombine
void Windows_Pfad_Verbinden(ddpstring *ret, ddpstring *a, ddpstring *b) {
	if (strlen(a->str) + 1 + strlen(b->str) + 1 >= DDP_MAX_WIN_PATH) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	char joined[DDP_MAX_WIN_PATH];
	if (!PathCombine(joined, a->str, b->str)) {
		// TODO: Error Handling
		*ret = DDP_EMPTY_STRING;
		return;
	}

	int len = strlen(joined) + 1;
	char *str = DDP_ALLOCATE(char, len);
	memcpy(str, joined, len);
	ret->str = str;
	ret->cap = len;
}
