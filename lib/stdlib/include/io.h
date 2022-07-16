#ifndef DDP_IO_H
#define DDP_IO_H

#ifdef _WIN32
#include <Windows.h>

HANDLE* get_stdin_handle();
#endif // _WIN32

// discards all characters in stdin up to and including '\n' or EOF
void flush_stdin();

#endif // DDP_IO_H