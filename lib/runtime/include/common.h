#ifndef DDP_COMMON_H
#define DDP_COMMON_H

#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// print the error message to stderr and exit with exit_code and calling end_runtime before exit
void runtime_error(int exit_code, const char* fmt, ...);

#endif // DDP_COMMON_H