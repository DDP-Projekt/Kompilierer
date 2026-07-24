#include "DDP/debug.h"
#include <stdarg.h>

void ddp_debug_log(const char *fmt __attribute__((__unused__)), ...) {
#ifdef DDP_DEBUG
	va_list argptr;
	va_start(argptr, fmt);

	DDP_DBGLOG(fmt, argptr)

	va_end(argptr);
#endif // DDP_DEBUG
}
