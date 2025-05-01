#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/error.h"
#include "DDP/winapi-path.h"
#include <stdbool.h>
#include <string.h>

// TODO: Use PathCchCanonicalize
void Windows_Saeubern(ddpstring *ret, ddpstring *path) {
	DDP_MIGHT_ERROR;
	if (path->len + 1 >= DDP_MAX_WIN_PATH) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	char cleaned[DDP_MAX_WIN_PATH];
	if (!PathCanonicalize(cleaned, DDP_STRING_DATA(path))) {
		ddp_error("Pfad konnte nicht gesäubert werden", false);
		*ret = DDP_EMPTY_STRING;
		return;
	}

	ddp_string_from_constant(ret, cleaned);
}

// TODO: Use PathCchCombine
void Windows_Pfad_Verbinden(ddpstring *ret, ddpstring *a, ddpstring *b) {
	DDP_MIGHT_ERROR;
	if (a->len + 1 + b->len + 1 >= DDP_MAX_WIN_PATH) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	char joined[DDP_MAX_WIN_PATH];
	if (!PathCombine(joined, DDP_STRING_DATA(a), DDP_STRING_DATA(b))) {
		ddp_error("Pfad konnte nicht verbunden werden", false);
		*ret = DDP_EMPTY_STRING;
		return;
	}

	ddp_string_from_constant(ret, joined);
}
