#ifndef DDP_COMMON_H
#define DDP_COMMON_H

#include <errno.h>
#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

// print the error message to stderr and exit with exit_code and calling end_runtime before exit
void ddp_runtime_error(int exit_code, const char *fmt, ...);
void print_backtrace(void);

#endif // DDP_COMMON_H
