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
	if (!PathCanonicalize(cleaned, DDP_GET_STRING_PTR(path))) {
		ddp_error("Pfad konnte nicht gesäubert werden", false);
		return;
	}

	ddp_string_from_constant(ret, cleaned);
}

// TODO: Use PathCchCombine
void Windows_Pfad_Verbinden(ddpstring *ret, ddpstring *a, ddpstring *b) {
	DDP_MIGHT_ERROR;
	*ret = DDP_EMPTY_STRING;
	if (ddp_strlen(a) + 1 + ddp_strlen(b) + 1 >= DDP_MAX_WIN_PATH) {
		return;
	}

	char joined[DDP_MAX_WIN_PATH];
	if (!PathCombine(joined, DDP_GET_STRING_PTR(a), DDP_GET_STRING_PTR(b))) {
		ddp_error("Pfad konnte nicht verbunden werden", false);
		return;
	}

	ddp_string_from_constant(ret, joined);
}
