#ifndef WINAPI_PATH_H
#define WINAPI_PATH_H

#define MAX_PATH 260
#include "ddptypes.h"
#include <string.h>


bool PathCanonicalize(char *buffer, const char *path);
char *PathCombine(char *lpszDest, const char *lpszDir, const char *lpszFile);

#endif // WINAPI_PATH_H
