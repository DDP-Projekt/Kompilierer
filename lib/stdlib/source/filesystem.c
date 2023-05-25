#include "ddptypes.h"
#include "ddpwindows.h"
#include "sys/types.h"

// copied from https://stackoverflow.com/questions/11238918/s-isreg-macro-undefined
#define _CRT_INTERNAL_NONSTDC_NAMES 1
#include "sys/stat.h"
#if !defined(S_ISDIR) && defined(S_IFMT) && defined(S_IFDIR)
  #define S_ISDIR(m) (((m) & S_IFMT) == S_IFDIR)
#endif

#ifdef DDPOS_WINDOWS
#include <io.h>
#include <direct.h>
#define access _access
#define F_OK 0
#define PATH_SEPERATOR "/\\"
#else
#include "unistd.h"
#define PATH_SEPERATOR "/"
#endif // DDPOS_WINDOWS

ddpbool Existiert_Datei(ddpstring* Pfad) {
	return access(Pfad->str, F_OK) == 0;
}

ddpbool Erstelle_Ordner(ddpstring* Pfad) {
	#ifdef DDPOS_LINUX
	#define _mkdir(arg) mkdir(arg, 0700)
#endif // DDPOS_LINUX

	// recursively create every directory needed to create the final one
	char* it = Pfad->str;
	while ((it = strpbrk(it, PATH_SEPERATOR)) != NULL) {
		*it = '\0';
		if (_mkdir(Pfad->str) != 0) return false;
		*it = '/';
		it++;
	}

	// == '/' because it might have already been created
	return Pfad->str[Pfad->cap - 2] == '/' || _mkdir(Pfad->str) == 0;

#ifdef DDPOS_LINUX
	#undef _mkdir
#endif // DDPOS_LINUX
}

ddpbool Lösche_Datei(ddpstring* Pfad) {
	return remove(Pfad->str) == 0;
}

ddpbool Ist_Ordner(ddpstring* Pfad) {
#ifdef DDPOS_WINDOWS
	#define stat _stat
#endif // DDPOS_DDPOSWINDOWS

	struct stat path_stat;
	if (stat(Pfad->str, &path_stat) != 0) return false;
	return S_ISDIR(path_stat.st_mode);
#ifdef DDPOS_WINDOWS
	#undef stat
#endif // DDPOS_WINDOWS
}

