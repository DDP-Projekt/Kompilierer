#ifndef DDP_IO_H
#define DDP_IO_H

#ifdef _WIN32
#include <Windows.h>

HANDLE* get_stdin_handle();
#endif // _WIN32

void flush_stdin();

#endif // DDP_IO_H