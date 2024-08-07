#ifndef WINAPI_PATH_H
#define WINAPI_PATH_H

#define DDP_MAX_WIN_PATH 260

#include <stdbool.h>

bool PathCanonicalize(char *buffer, const char *path);
char *PathCombine(char *lpszDest, const char *lpszDir, const char *lpszFile);

#endif // WINAPI_PATH_H
