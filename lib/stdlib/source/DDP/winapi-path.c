#include "DDP/winapi-path.h"
#include <string.h>

/*
 * Path Functions
 *
 * Copyright 1999, 2000 Juergen Schmied
 * Copyright 2001, 2002 Jon Griffiths
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA 02110-1301, USA
 */

// Adapted slightly to bring it more inline with the unix api

/*
    PathCanonicalize
*/

char *CharNext(const char *ptr) {
	if (!*ptr) {
		return (char *)ptr;
	}
	//if (IsDBCSLeadByte(ptr[0]) && ptr[1]) return (char*)(ptr + 2);
	return (char *)(ptr + 1);
}

int PathIsUNCServerShare(const char *path) {
	bool seen_slash = 0;

	if (!path || *path++ != '\\' || *path++ != '\\') {
		return false;
	}

	while (*path) {
		if (*path == '\\') {
			if (seen_slash) {
				return false;
			}
			seen_slash = true;
		}

		path = CharNext(path);
	}

	return seen_slash;
}

bool PathCanonicalize(char *buffer, const char *path) {
	const char *src = path;
	char *dst = buffer;

	if (dst) {
		*dst = '\0';
	}

	if (!dst || !path) {
		return 0;
	}

	if (!*path) {
		*buffer++ = '\\';
		*buffer = '\0';
		return 1;
	}

	// Copy path root
	if (*src == '\\') {
		*dst++ = *src++;
	} else if (*src && src[1] == ':') {
		/* X:\ */
		*dst++ = *src++;
		*dst++ = *src++;
		if (*src == '\\') {
			*dst++ = *src++;
		}
	}

	// Canonicalize the rest of the path
	while (*src) {
		if (*src != '.') {
			*dst++ = *src++;
			continue;
		}

		// Handle *\.
		if (src[-1] == '\\' && src[1] == '\0') {
			dst--; /* Remove \ */
			src++; /* Skip . */
		}
		// Handle *.\  or  .\*  or  *\.\*  or  *:.\*
		else if (src[1] == '\\' && (src == path || src[-1] == '\\' || src[-1] == ':')) {
			src += 2; /* Skip .\ */
		}
		// Handle *\..*
		else if (src[1] == '.' && dst != buffer && dst[-1] == '\\') { //
			/* 
                \.. backs up a directory, over the root if it has no \ following X:.
                * .. is ignored if it would remove a UNC server name or initial \\
            */
			if (dst != buffer) {
				*dst = '\0'; // Allow PathIsUNCServerShareA test on lpszBuf
				if (dst > buffer + 1 && dst[-1] == '\\' && (dst[-2] != '\\' || dst > buffer + 2)) {
					if (dst[-2] == ':' && (dst > buffer + 3 || dst[-3] == ':')) {
						dst -= 2;
						while (dst > buffer && *dst != '\\') {
							dst--;
						}

						if (*dst == '\\') {
							dst++; /* Reset to last '\' */
						} else {
							dst = buffer; // Start path again from new root
						}
					} else if (dst[-2] != ':' && !PathIsUNCServerShare(buffer)) {
						dst -= 2;
					}
				}
				while (dst > buffer && *dst != '\\') {
					dst--;
				}
				if (dst == buffer) {
					*dst++ = '\\';
					src++;
				}
			}
			src += 2; // Skip .. in src path
		}
		// start of path
		else if (dst == buffer) {
			// Handle ..
			if (src[1] == '.') {
				src++;
			}
			// Handle .
			else if (src[1] == '\0') {
				*dst++ = '\\';
				src++;
			}
		} else {
			*dst++ = *src++;
		}
	}

	// Append \ to naked drive specs
	if (dst - buffer == 2 && dst[-1] == ':') {
		*dst++ = '\\';
	}

	*dst++ = '\0';
	return 1;
}

/*
    PathCombine
*/

char *lstrcpynA(char *lpString1, const char *lpString2, int iMaxLength) {
	char *d = lpString1;
	const char *s = lpString2;
	unsigned int count = iMaxLength;
	char *Ret = NULL;

	while ((count > 1) && *s) {
		count--;
		*d++ = *s++;
	}

	if (count) {
		*d = 0;
	}

	Ret = lpString1;

	return Ret;
}

char *PathAddBackslashA(char *lpszPath) {
	size_t iLen;
	char *prev = lpszPath;

	if (!lpszPath || (iLen = strlen(lpszPath)) >= DDP_MAX_WIN_PATH) {
		return NULL;
	}

	if (iLen) {
		do {
			lpszPath = CharNext(prev);
			if (*lpszPath) {
				prev = lpszPath;
			}
		} while (*lpszPath);

		if (*prev != '\\') {
			*lpszPath++ = '\\';
			*lpszPath = '\0';
		}
	}
	return lpszPath;
}

bool PathIsRootA(const char *lpszPath) {
	if (lpszPath && *lpszPath) {
		if (*lpszPath == '\\') {
			if (!lpszPath[1]) {
				return true; /* \ */
			} else if (lpszPath[1] == '\\') {
				bool bSeenSlash = false;
				lpszPath += 2;

				/* Check for UNC root path */
				while (*lpszPath) {
					if (*lpszPath == '\\') {
						if (bSeenSlash) {
							return false;
						}
						bSeenSlash = true;
					}
					lpszPath = CharNext(lpszPath);
				}
				return true;
			}
		} else if (lpszPath[1] == ':' && lpszPath[2] == '\\' && lpszPath[3] == '\0') {
			return true; /* X:\ */
		}
	}
	return false;
}

bool PathRemoveFileSpecA(char *lpszPath) {
	char *lpszFileSpec = lpszPath;
	bool bModified = false;

	if (lpszPath) {
		/* Skip directory or UNC path */
		if (*lpszPath == '\\') {
			lpszFileSpec = ++lpszPath;
		}
		if (*lpszPath == '\\') {
			lpszFileSpec = ++lpszPath;
		}

		while (*lpszPath) {
			if (*lpszPath == '\\') {
				lpszFileSpec = lpszPath; /* Skip dir */
			} else if (*lpszPath == ':') {
				lpszFileSpec = ++lpszPath; /* Skip drive */
				if (*lpszPath == '\\') {
					lpszFileSpec++;
				}
			}
			if (!(lpszPath = CharNext(lpszPath))) {
				break;
			}
		}

		if (*lpszFileSpec) {
			*lpszFileSpec = '\0';
			bModified = true;
		}
	}
	return bModified;
}

bool PathStripToRootA(char *lpszPath) {

	if (!lpszPath) {
		return false;
	}
	while (!PathIsRootA(lpszPath)) {
		if (!PathRemoveFileSpecA(lpszPath)) {
			return false;
		}
	}
	return true;
}

char *PathCombine(char *lpszDest, const char *lpszDir, const char *lpszFile) {
	char szTemp[DDP_MAX_WIN_PATH];
	bool bUseBoth = false, bStrip = false;

	/* Invalid parameters */
	if (!lpszDest) {
		return NULL;
	}
	if (!lpszDir && !lpszFile) {
		lpszDest[0] = 0;
		return NULL;
	}

	if ((!lpszFile || !*lpszFile) && lpszDir) {
		/* Use dir only */
		lstrcpynA(szTemp, lpszDir, DDP_MAX_WIN_PATH);
	} else if (!lpszDir || !*lpszDir || *lpszFile == '\\' || (*lpszFile && lpszFile[1] == ':')) {
		if (!lpszDir || !*lpszDir || *lpszFile != '\\' || (lpszFile && (lpszFile[0] == '\\') && (lpszFile[1] == '\\'))) {
			/* Use file only */
			lstrcpynA(szTemp, lpszFile, DDP_MAX_WIN_PATH);
		} else {
			bUseBoth = true;
			bStrip = true;
		}
	} else {
		bUseBoth = true;
	}

	if (bUseBoth) {
		lstrcpynA(szTemp, lpszDir, DDP_MAX_WIN_PATH);
		if (bStrip) {
			PathStripToRootA(szTemp);
			lpszFile++; /* Skip '\' */
		}
		if (!PathAddBackslashA(szTemp) || strlen(szTemp) + strlen(lpszFile) >= DDP_MAX_WIN_PATH) {
			lpszDest[0] = 0;
			return NULL;
		}
		strcat(szTemp, lpszFile);
	}

	PathCanonicalize(lpszDest, szTemp);
	return lpszDest;
}
