#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/error.h"
#include "DDP/winapi-path.h"
#include <stdbool.h>
#include <string.h>

// TODO: Use PathCchCanonicalize
void Windows_Saeubern(ddpstring *ret, ddpstring *path) {
	DDP_MIGHT_ERROR;
	*ret = DDP_EMPTY_STRING;
	if (ddp_strlen(path) + 1 >= DDP_MAX_WIN_PATH) {
		return;
	}

	char cleaned[DDP_MAX_WIN_PATH];
	if (!PathCanonicalize(cleaned, path->str)) {
		ddp_error("Pfad konnte nicht gesäubert werden", false);
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
	DDP_MIGHT_ERROR;
	*ret = DDP_EMPTY_STRING;
	if (ddp_strlen(a) + 1 + ddp_strlen(b) + 1 >= DDP_MAX_WIN_PATH) {
		return;
	}

	char joined[DDP_MAX_WIN_PATH];
	if (!PathCombine(joined, a->str, b->str)) {
		ddp_error("Pfad konnte nicht verbunden werden", false);
		return;
	}

	int len = strlen(joined) + 1;
	char *str = DDP_ALLOCATE(char, len);
	memcpy(str, joined, len);
	ret->str = str;
	ret->cap = len;
}
