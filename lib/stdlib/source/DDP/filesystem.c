#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/ddpwindows.h"
#include "DDP/debug.h"
#include "DDP/error.h"
#include "DDP/utf8/utf8.h"
#include <ctype.h>
#include <dirent.h>
#include <fcntl.h>
#include <libgen.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <time.h>

// copied from https://stackoverflow.com/questions/11238918/s-isreg-macro-undefined
// to handle missing macros on Windows
#define _CRT_INTERNAL_NONSTDC_NAMES 1
#include <sys/stat.h>
#if !defined(S_ISDIR) && defined(S_IFMT) && defined(S_IFDIR)
#define S_ISDIR(m) (((m) & S_IFMT) == S_IFDIR)
#endif

#ifdef DDPOS_WINDOWS
#include <WinBase.h>
#include <direct.h>
#include <io.h>

#define access _access
// #define stat _stat
#define mkdir _mkdir
#define F_OK 0
#define PATH_SEPERATOR "/\\" // set of possible seperators
#else
#include <fcntl.h>
#include <unistd.h>

#define PATH_SEPERATOR "/" // set of possible seperators
#define mkdir(arg) mkdir(arg, 0700)
#endif // DDPOS_WINDOWS

ddpbool Existiert_Pfad(ddpstring *Pfad) {
	if (ddp_string_empty(Pfad)) {
		return false;
	}
	return access(Pfad->str, F_OK) == 0;
}

ddpbool Erstelle_Ordner(ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	if (ddp_string_empty(Pfad)) {
		ddp_error("Der gegebene Pfad ist leer", false);
		return false;
	}
	// recursively create every directory needed to create the final one
	char *it = Pfad->str;
	while ((it = strpbrk(it, PATH_SEPERATOR)) != NULL) {
		*it = '\0';
		if (mkdir(Pfad->str) != 0 && errno != EEXIST) {
			ddp_error("Fehler beim Erstellen des Ordners '" DDP_STRING_FMT "': ", true, Pfad->str);
			return false;
		}
		*it = '/';
		it++;
	}

	// == '/' because it might have already been created
	if (Pfad->str[Pfad->cap - 2] == '/') {
		return true;
	} else if (mkdir(Pfad->str) != 0 && errno != EEXIST) {
		ddp_error("Fehler beim Erstellen des Ordners '" DDP_STRING_FMT "': ", true, Pfad->str);
		return false;
	}
	return true;
}

ddpbool Ist_Ordner(ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	if (ddp_string_empty(Pfad)) {
		return false;
	}

	// remove possible trailing seperators
	char *it = Pfad->str + Pfad->cap - 2; // last character in str
	while (it >= Pfad->str) {
		if (strpbrk(it--, PATH_SEPERATOR) != NULL) {
			*(it + 1) = '\0';
		} else {
			break;
		}
	}

	struct stat path_stat;
	if (stat(Pfad->str, &path_stat) != 0) {
		ddp_error("Fehler beim Überprüfen des Pfades '" DDP_STRING_FMT "': ", true, Pfad->str);
		return false;
	}
	return S_ISDIR(path_stat.st_mode);
}

// copied from https://stackoverflow.com/questions/2256945/removing-a-non-empty-directory-programmatically-in-c-or-c
static int remove_directory(const char *path) {
	DDP_MIGHT_ERROR;

	DIR *d = opendir(path);
	size_t path_len = strlen(path);
	int r = -1;

	if (d) {
		struct dirent *p;
		r = 0;
		while (!r && (p = readdir(d))) {
			int r2 = -1;
			char *buf;
			size_t len;

			// Skip the names "." and ".." as we don't want to recurse on them.
			if (!strcmp(p->d_name, ".") || !strcmp(p->d_name, "..")) {
				continue;
			}

			len = path_len + strlen(p->d_name) + 2;
			buf = DDP_ALLOCATE(char, len);

			if (buf) {
				struct stat statbuf;

				snprintf(buf, len, "" DDP_STRING_FMT "/" DDP_STRING_FMT "", path, p->d_name);
				if (!stat(buf, &statbuf)) {
					// technically, this is a TOCTOU vulnerability (see https://cwe.mitre.org/data/definitions/367.html)
					// but as the unlinkat function is not present in mingw, this is not easily resolvable (or might even be impossible)
					// to make it short: this might be vulnerable in edge-cases, but I don't care
					if (S_ISDIR(statbuf.st_mode)) {
						r2 = remove_directory(buf);
					} else {
						r2 = unlink(buf);
					}
				}
				DDP_FREE(char, buf);
			}
			r = r2;
		}
		closedir(d);
	} else {
		ddp_error("Fehler beim Öffnen des Ordners '" DDP_STRING_FMT "': ", true, path);
		return -1;
	}

	if (!r) {
		r = rmdir(path);
	} else {
		ddp_error("Fehler beim Löschen des Ordners '" DDP_STRING_FMT "': ", true, path);
		return -1;
	}

	return r;
}

ddpbool Loesche_Pfad(ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	if (ddp_string_empty(Pfad)) {
		ddp_error("Der gegebene Pfad ist leer", false);
		return false;
	}

	if (Ist_Ordner(Pfad)) {
		return remove_directory(Pfad->str) == 0;
	}
	if (unlink(Pfad->str) != 0) {
		ddp_error("Fehler beim Löschen des Pfades '" DDP_STRING_FMT "': ", true, Pfad->str);
		return false;
	}
	return true;
}

ddpbool Pfad_Verschieben(ddpstring *Pfad, ddpstring *NeuerName) {
	DDP_MIGHT_ERROR;

	if (ddp_string_empty(Pfad)) {
		ddp_error("Der gegebene Pfad ist leer", false);
		return false;
	}

	struct stat path_stat;
	if (stat(NeuerName->str, &path_stat) != 0) {
		ddp_error("Fehler beim Überprüfen des Pfades '" DDP_STRING_FMT "': ", true, NeuerName->str);
		return false;
	}

	// https://stackoverflow.com/questions/64276902/mv-command-implementation-in-c-not-moving-files-to-different-directory
	if (S_ISDIR(path_stat.st_mode)) {
		char *path_copy = DDP_ALLOCATE(char, Pfad->cap);
		memcpy(path_copy, Pfad->str, Pfad->cap);

		char *base = basename(path_copy);
		size_t len_base = strlen(base);

		NeuerName->str = DDP_GROW_ARRAY(char, NeuerName->str, NeuerName->cap, NeuerName->cap + len_base + 1);
		strcat(NeuerName->str, "/");
		strcat(NeuerName->str, base);
		NeuerName->cap = NeuerName->cap + len_base + 1;

		DDP_FREE(char, path_copy);
	}
	if (rename(Pfad->str, NeuerName->str) != 0) {
		ddp_error("Fehler beim Verschieben des Pfades '" DDP_STRING_FMT "' nach '" DDP_STRING_FMT "': ", true, Pfad->str, NeuerName->str);
		return false;
	}
	return true;
}

static void formatDateStr(ddpstring *str, struct tm *time) {
	// make string empty
	*str = DDP_EMPTY_STRING;

	// format string
	char buff[30];
	int size = sprintf(buff, "%02d:%02d:%02d %02d.%02d.%02d", time->tm_hour, time->tm_min, time->tm_sec, time->tm_mday, time->tm_mon + 1, time->tm_year + 1900);

	str->cap = size + 1;
	str->str = DDP_ALLOCATE(char, str->cap);
	strcpy(str->str, buff);
	str->str[str->cap - 1] = '\0';
}

void Zugriff_Datum(ddpstring *ret, ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	struct stat st;
	if (stat(Pfad->str, &st) != 0) {
		*ret = (ddpstring){0};
		ddp_error("Fehler beim Überprüfen des Pfades '" DDP_STRING_FMT "': ", true, Pfad->str);
		return;
	}
	struct tm *tm = localtime(&st.st_atime);

	formatDateStr(ret, tm);
}

void AEnderung_Datum(ddpstring *ret, ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	struct stat st;
	if (stat(Pfad->str, &st) != 0) {
		*ret = (ddpstring){0};
		ddp_error("Fehler beim Überprüfen des Pfades '" DDP_STRING_FMT "': ", true, Pfad->str);
		return;
	}
	struct tm *tm = localtime(&st.st_mtime);

	formatDateStr(ret, tm);
}

void Status_Datum(ddpstring *ret, ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	struct stat st;
	if (stat(Pfad->str, &st) != 0) {
		*ret = (ddpstring){0};
		ddp_error("Fehler beim Überprüfen des Pfades '" DDP_STRING_FMT "': ", true, Pfad->str);
		return;
	}
	struct tm *tm = localtime(&st.st_ctime);

	formatDateStr(ret, tm);
}

ddpint Datei_Groesse(ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	struct stat st;
	if (stat(Pfad->str, &st) != 0) {
		ddp_error("Fehler beim Überprüfen des Pfades '" DDP_STRING_FMT "': ", true, Pfad->str);
		return -1;
	}

	return (ddpint)st.st_size;
}

ddpint Datei_Modus(ddpstring *Pfad) {
	DDP_MIGHT_ERROR;

	struct stat st;
	if (stat(Pfad->str, &st) != 0) {
		ddp_error("Fehler beim Überprüfen des Pfades '" DDP_STRING_FMT "': ", true, Pfad->str);
		return -1;
	}

	return (ddpint)st.st_mode;
}

// UNIX: https://stackoverflow.com/questions/2180079/how-can-i-copy-a-file-on-unix-using-c
ddpbool Datei_Kopieren(ddpstring *Pfad, ddpstring *Kopiepfad) {
	DDP_MIGHT_ERROR;

	if (ddp_string_empty(Pfad)) {
		ddp_error("Der gegebene Pfad ist leer", false);
		return -1;
	}

#ifdef DDPOS_WINDOWS
	return (ddpbool)CopyFile(Pfad->str, Kopiepfad->str, false);
#else  // DDPOW_LINUX
	if (!Pfad->str || !Kopiepfad->str) {
		return (ddpbool) false;
	}

	int fd_to, fd_from;

	if ((fd_from = open(Pfad->str, O_RDONLY)) < 0) {
		return (ddpbool) false;
	}
	if ((fd_to = open(Kopiepfad->str, O_WRONLY | O_CREAT | O_TRUNC, 0666)) < 0) {
		goto out_error;
	}

	char buff[1 << 13]; // 8KB
	ssize_t nread;
	while ((nread = read(fd_from, buff, sizeof(buff))) > 0) {
		char *out_ptr = buff;
		do {
			ssize_t nwritten = write(fd_to, out_ptr, nread);

			if (nwritten >= 0) {
				nread -= nwritten;
				out_ptr += nwritten;
			} else if (errno != EINTR) {
				goto out_error;
			}
		} while (nread > 0);
	}

	if (nread == 0) {
		// TODO: check this
		struct stat from_info;
		if (fstat(fd_from, &from_info) >= 0) {
			fchmod(fd_to, from_info.st_mode);
		}

		close(fd_from);
		close(fd_to);
		return (ddpbool) true;
	}

out_error:
	close(fd_from);
	if (fd_to >= 0) {
		close(fd_to);
	}

	return (ddpbool) false;
#endif // DDPOS_WINDOWS
}

ddpint Lies_Text_Datei(ddpstring *Pfad, ddpstringref ref) {
	DDP_MIGHT_ERROR;

	FILE *file = fopen(Pfad->str, "r");
	if (file) {
		fseek(file, 0, SEEK_END);			  // seek the last byte in the file
		size_t string_size = ftell(file) + 1; // file_size + '\0'
		rewind(file);						  // go back to file start
		ref->str = ddp_reallocate(ref->str, ref->cap, string_size);
		ref->cap = string_size;
		size_t read = fread(ref->str, sizeof(char), string_size - 1, file);
		fclose(file);
		ref->str[ref->cap - 1] = '\0';
		if (read != string_size - 1) {
			ddp_error("Fehler beim Lesen der Datei '" DDP_STRING_FMT "': ", true, Pfad->str);
		}
		return (ddpint)read;
	}
	ddp_error("Fehler beim Öffnen der Datei '" DDP_STRING_FMT "': ", true, Pfad->str);
	return -1;
}

ddpint Schreibe_Text_Datei(ddpstring *Pfad, ddpstring *text) {
	DDP_MIGHT_ERROR;

	FILE *file = fopen(Pfad->str, "w");
	if (file) {
		int ret = fprintf(file, text->str);
		fclose(file);
		if (ret < 0) {
			ddp_error("Fehler beim Schreiben der Datei '" DDP_STRING_FMT "': ", true, Pfad->str);
			return ret;
		}
		return (ddpint)ret;
	}
	ddp_error("Fehler beim Öffnen der Datei '" DDP_STRING_FMT "': ", true, Pfad->str);
	return -1;
}

/* Generalized buffered File IO */

#define DDP_MAX(a, b) ((a) > (b) ? (a) : (b))
#define DDP_MIN(a, b) ((a) < (b) ? (a) : (b))

#define BUFFER_SIZE 4095

#define MODUS_NUR_LESEN 1
#define MODUS_NUR_SCHREIBEN 2
#define MODUS_LESEN_SCHREIBEN 4

#define MODUS_ERSTELLEN 8
#define MODUS_ANHAENGEN 16
#define MODUS_TRUNKIEREN 32

typedef struct {
	int fd;					// posix file descriptor
	ddpint id;				// unique id for this file
	ddpstring path;			// path to the file (which it was opened with)
	ddpint mode;			// (ddp) mode the file was opened with
	bool eof;				// whether the file has reached the end
	ddpstring read_buffer;	// buffer for reading
	ddpstring write_buffer; // buffer for writing
	int read_buffer_start;	// start index of the read buffer
	int read_buffer_end;	// end index of the read buffer
	int write_index;		// index of the write buffer
} InternalFile;

static struct {
	InternalFile *files;
	int cap;
} open_files = {0};

static ddpint cur_file_id = 1;

// returns a fresh InternalFile
static InternalFile new_internal_file(void) {
	InternalFile file = {0};
	file.fd = -1;
	return file;
}

// zeros out an already created/used InternalFile
static void zero_internal_file(InternalFile *file) {
	DDP_DBGLOG("zero_internal_file(%p, %d, " DDP_INT_FMT ")", file, file->fd, file->id);
	ddp_free_string(&file->path);
	ddp_free_string(&file->read_buffer);
	ddp_free_string(&file->write_buffer);
	*file = (InternalFile){0};
	file->fd = -1;
}

static int flush_write_buffer(InternalFile *file);

// atexit handler to free all open files
static void free_open_files(void) {
	DDP_DBGLOG("free_open_files()");
	for (int i = 0; i < open_files.cap; i++) {
		InternalFile *file = &open_files.files[i];
		if (file->fd >= 0) {
			flush_write_buffer(file);
			if (close(file->fd) < 0) {
				// no need for error handling in atexit
			}
		}
		zero_internal_file(file);
	}
	DDP_DBGLOG("free_open_files() - freeing files array");
	DDP_FREE_ARRAY(InternalFile, open_files.files, open_files.cap);
}

// searches for the first unused file descriptor and returns it
// if no file descriptor is available, it will grow the array and return the new index
static ddpint add_internal_file(void) {
	DDP_DBGLOG("add_internal_file()");
	// search for the first unused file descriptor
	for (int i = 0; i < open_files.cap; i++) {
		if (open_files.files[i].fd == -1) {
			zero_internal_file(&open_files.files[i]);
			return i;
		}
	}

	// if no files were allocated yet, register the atexit function to free the files array on program exit
	if (open_files.files == NULL) {
		atexit(free_open_files);
	}

	// if no file descriptor is available, grow the array and return the new index
	DDP_DBGLOG("add_internal_file() - growing files array");
	ddpint old_cap = open_files.cap;
	open_files.cap = DDP_GROW_CAPACITY(open_files.cap);
	open_files.files = DDP_GROW_ARRAY(InternalFile, open_files.files, old_cap, open_files.cap);

	// the new files are initialized to 0
	for (int i = old_cap; i < open_files.cap; i++) {
		open_files.files[i] = new_internal_file();
	}

	return old_cap;
}

static InternalFile *get_internal_file(ddpint index, ddpint id) {
	// check if the index is valid
	if (index < 0 || index >= open_files.cap) {
		ddp_error("Die Datei ist nicht geöffnet", false);
		return NULL;
	}
	InternalFile *file = &open_files.files[index];
	// check if the id is valid
	// if the id is not valid, a copy of the file was probably used after it was already closed
	if (file->id != id) {
		ddp_error("Die Datei ist nicht mehr geöffnet", false);
		return NULL;
	}
	return file;
}

// closes the file descriptor and zeros out the file
static void close_internal_file(ddpint index) {
	if (index < 0) {
		ddp_error("Nicht offene Datei kann nicht geschlossen werden", false);
		return;
	}

	InternalFile *file = &open_files.files[index];

	flush_write_buffer(file);

	// attempt to close the file and report possible errors (very rare)
	if (file->fd >= 0) {
		if (close(file->fd) < 0) {
			ddp_error("Fehler beim Schließen der Datei '" DDP_STRING_FMT "': ", true, file->path.str);
		}
	}

	zero_internal_file(file);
}

static int to_posix_mode(ddpint Modus) {
	int ret = 0;
	if ((Modus & MODUS_NUR_LESEN) == MODUS_NUR_LESEN) {
		ret |= O_RDONLY;
	} else if ((Modus & MODUS_NUR_SCHREIBEN) == MODUS_NUR_SCHREIBEN) {
		ret |= O_WRONLY;
	} else if ((Modus & MODUS_LESEN_SCHREIBEN) == MODUS_LESEN_SCHREIBEN) {
		ret |= O_RDWR;
	}

	if ((Modus & MODUS_ERSTELLEN) == MODUS_ERSTELLEN) {
		ret |= O_CREAT;
	}
	if ((Modus & MODUS_ANHAENGEN) == MODUS_ANHAENGEN) {
		ret |= O_APPEND;
	}
	if ((Modus & MODUS_TRUNKIEREN) == MODUS_TRUNKIEREN) {
		ret |= O_TRUNC;
	}

	return ret;
}

static bool is_read_mode(ddpint Modus) {
	return (Modus & (MODUS_NUR_LESEN | MODUS_LESEN_SCHREIBEN)) != 0;
}

static bool is_write_mode(ddpint Modus) {
	return (Modus & (MODUS_NUR_SCHREIBEN | MODUS_LESEN_SCHREIBEN)) != 0;
}

static int read_buff_len(InternalFile *file) {
	return file->read_buffer_end - file->read_buffer_start;
}

static char *read_buff_start_ptr(InternalFile *file) {
	return file->read_buffer.str + file->read_buffer_start;
}

// points to the first byte after the last byte in the buffer
static char *read_buff_end_ptr(InternalFile *file) {
	return file->read_buffer.str + file->read_buffer_end;
}

// clears the read buffer and reads in the next chunk
// returns the number of bytes read (Lese_Puffer_Ende)
// 0 means EOF and -1 means error
static int fill_read_buffer(InternalFile *file) {
	// validate that the file can be read from
	if (!is_read_mode(file->mode)) {
		ddp_error("Fehler beim Lesen der Datei '" DDP_STRING_FMT "': Die Datei wurde nicht zum Lesen geöffnet", false, file->path.str);
		return -1;
	}
	if (file->eof) {
		ddp_error("Fehler beim Lesen der Datei '" DDP_STRING_FMT "': Das Ende der Datei ist erreicht", false, file->path.str);
		return -1;
	}

	// allocate the buffer if needed
	if (file->read_buffer.cap < BUFFER_SIZE) {
		file->read_buffer.str = DDP_ALLOCATE(char, BUFFER_SIZE + 1);
		file->read_buffer.cap = BUFFER_SIZE + 1;
		file->read_buffer.str[BUFFER_SIZE] = '\0';
	}

	// read in the next chunk
	int ret = read(file->fd, file->read_buffer.str, BUFFER_SIZE);
	file->read_buffer_start = 0;
	file->read_buffer_end = ret;
	// important to null-terminate the buffer if less than BUFFER_SIZE bytes were read
	file->read_buffer.str[ret] = '\0';

	if (ret < 0) {
		ddp_error("Fehler beim Lesen der Datei '" DDP_STRING_FMT "': ", true, file->path.str);
		file->read_buffer_end = 0;
	} else if (ret == 0) {
		file->eof = true;
	}

	return ret;
}

// validate that the file can be written to
#define DDP_ENSURE_WRITE_MODE(file)                                                                                                                \
	if (!is_write_mode(file->mode)) {                                                                                                              \
		ddp_error("Fehler beim Schreiben in die Datei '" DDP_STRING_FMT "': Die Datei wurde nicht zum Schreiben geöffnet", false, file->path.str); \
		return;                                                                                                                                    \
	}

static int flush_write_buffer(InternalFile *file) {
	if (file->write_index == 0) {
		return 0;
	}

	int ret = write(file->fd, file->write_buffer.str, file->write_index);
	if (ret < 0) {
		ddp_error("Fehler beim Schreiben der Datei '" DDP_STRING_FMT "': ", true, file->path.str);
		return -1;
	}

	file->write_index = 0;
	return ret;
}

static int write_buffer_space(InternalFile *file) {
	return file->write_buffer.cap - 1 - file->write_index;
}

static int write_to_buffer(InternalFile *file, const char *data, ddpint len) {
	DDP_DBGLOG("write_to_buffer(%p, %p, " DDP_INT_FMT ")", file, data, len);
	// allocate the buffer if needed
	if (file->write_buffer.cap < BUFFER_SIZE) {
		file->write_buffer.str = DDP_ALLOCATE(char, BUFFER_SIZE + 1);
		file->write_buffer.cap = BUFFER_SIZE + 1;
		file->write_buffer.str[BUFFER_SIZE] = '\0';
	}

	while (len > 0) {
		int space = write_buffer_space(file);
		if (space == 0) {
			if (flush_write_buffer(file) < 0) {
				return -1;
			}
			space = write_buffer_space(file);
		}
		space = DDP_MIN(space, len);
		memcpy(file->write_buffer.str + file->write_index, data, space);
		file->write_index += space;
		data += space;
		len -= space;
	}

	return len;
}

typedef struct {
	ddpint index;
	ddpint id;
} Datei;

typedef Datei *DateiRef;

void Datei_Oeffnen(DateiRef datei, ddpstring *Pfad, ddpint Modus) {
	DDP_MIGHT_ERROR;
	DDP_DBGLOG("Datei_Oeffnen(%p, %p, " DDP_INT_FMT ")", datei, Pfad, Modus);

	// create a new file and store the index in the DateiRef
	datei->index = add_internal_file();
	InternalFile *file = &open_files.files[datei->index];

	// store other information in the file
	file->path = *Pfad;
	*Pfad = DDP_EMPTY_STRING;
	file->mode = Modus;

	// actually open the file
	int flags = to_posix_mode(Modus);
#ifdef DDPOS_WINDOWS
	// open files in binary mode on windows, to not mess with line endings
	// not needed (even mostly non-existent) on unix
	flags |= O_BINARY;
#endif // DDPOS_WINDOWS
	if ((file->fd = open(file->path.str, flags, 0666)) < 0) {
		close_internal_file(datei->index);
		ddp_error("Fehler beim Öffnen der Datei '" DDP_STRING_FMT "': ", true, file->path.str);
		datei->index = -1;
	}
	// add validation information
	file->id = cur_file_id++;
	datei->id = file->id;
}

void Datei_Schliessen(DateiRef datei) {
	DDP_MIGHT_ERROR;
	DDP_DBGLOG("Datei_Schliessen(%p)", datei);

	// close the file and do some cleanup
	close_internal_file(datei->index);
	datei->index = -1;
	datei->id = -1;
}

void Datei_Lies_N_Zeichen(ddpstring *ret, DateiRef datei, ddpint N) {
	DDP_MIGHT_ERROR;

	*ret = DDP_EMPTY_STRING;
	// get the internal file and validate it
	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return;
	}

	// early return if N is not positive
	if (N <= 0) {
		return;
	}

	// preallocate the string
	ret->str = DDP_ALLOCATE(char, N + 1);
	ret->cap = N + 1;
	ret->str[N] = '\0';

	// read buffer-sized chunks until N characters are read
	for (ddpint copied = 0; copied < N;) {
		if (read_buff_len(file) == 0) {
			int read = fill_read_buffer(file);
			if (read < 0) {
				return;
			} else if (read == 0) {
				// EOF was reached so we need to move the null-terminator to the actual end of the string
				ret->str[copied] = '\0';
				return;
			}
		}

		// copy the next chunk
		ddpint copy_amount = DDP_MIN(read_buff_len(file), N - copied);
		memcpy(ret->str + copied, read_buff_start_ptr(file), copy_amount);
		file->read_buffer_start += copy_amount;
		copied += copy_amount;
	}
}

void Datei_Lies_Alles(ddpstring *ret, DateiRef datei) {
	DDP_MIGHT_ERROR;
	DDP_DBGLOG("Datei_Lies_Alles(%p)", datei);

	*ret = DDP_EMPTY_STRING;
	// get the internal file and validate it
	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return;
	}

	struct stat file_info;
	if (fstat(file->fd, &file_info) != 0) {
		ddp_error("Fehler beim Aufrufen von Datei Informationen: ", true);
		return;
	}

	Datei_Lies_N_Zeichen(ret, datei, file_info.st_size);
}

#define DDP_FILL_READ_BUFFER_OR_RETURN_BREAK(file) \
	if (read_buff_len(file) == 0) {                \
		int read = fill_read_buffer(file);         \
		if (read < 0) {                            \
			return;                                \
		} else if (read == 0) {                    \
			break;                                 \
		}                                          \
	}

void Datei_Lies_Zeile(ddpstring *ret, DateiRef datei) {
	DDP_MIGHT_ERROR;

	*ret = DDP_EMPTY_STRING;
	// get the internal file and validate it
	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return;
	}

	// read buffer-sized chunks until a newline is found
	while (!file->eof) {
		DDP_FILL_READ_BUFFER_OR_RETURN_BREAK(file);

		// search for the next newline
		char *newline = memchr(read_buff_start_ptr(file), '\n', read_buff_len(file));
		if (newline) {
			const size_t copy_amount = newline - read_buff_start_ptr(file) + 1;

			// allocate space for the string + '\n' (without '\0'!)
			ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap + copy_amount);
			// copy the string and the newline
			memcpy(ret->str + ret->cap, read_buff_start_ptr(file), copy_amount);
			ret->cap += copy_amount;

			DDP_DBGLOG("newline found: " DDP_INT_FMT ", %llu", ret->cap, copy_amount);
#ifdef DDPOS_WINDOWS
			// check windows line endings
			if (ret->str[ret->cap - 2] == '\r') {
				DDP_DBGLOG("carriage return found: " DDP_INT_FMT, ret->cap);
				// the carriage return becomes the null terminator
				ret->str[ret->cap - 2] = '\0';
				ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap - 1);
				ret->cap--;
			} else {
#endif
				// the newline becomes the null terminator
				ret->str[ret->cap - 1] = '\0';
#ifdef DDPOS_WINDOWS
			}
#endif

			file->read_buffer_start += copy_amount; // consume the newline
			return;
		} else {
			// no newline found, copy the whole buffer
			const size_t copy_amount = read_buff_len(file);

			// allocate space for the string (without '\0'!)
			ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap + copy_amount);
			// copy the string
			memcpy(ret->str + ret->cap, read_buff_start_ptr(file), copy_amount);
			ret->cap += copy_amount;

			file->read_buffer_start += copy_amount; // consume the whole buffer
		}
	}
	// EOF reached
	ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap + 1);
	ret->str[ret->cap] = '\0';
	ret->cap++;
}

void Datei_Lies_Wort(ddpstring *ret, DateiRef datei) {
	DDP_MIGHT_ERROR;

	*ret = DDP_EMPTY_STRING;
	// get the internal file and validate it
	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return;
	}

	// skip leading whitespace
	while (!file->eof) {
		DDP_FILL_READ_BUFFER_OR_RETURN_BREAK(file);

		bool found = false;
		// search for the next non-whitespace character
		const char *read_buff_end = read_buff_end_ptr(file);
		for (char *it = read_buff_start_ptr(file); it < read_buff_end; it++) {
			if (!isspace(*it)) {
				// found the start of the word
				file->read_buffer_start += it - read_buff_start_ptr(file);
				found = true;
				break;
			}
		}
		if (found) {
			break;
		}
		file->read_buffer_start = file->read_buffer_end; // consume the whole buffer
	}

	if (file->eof) {
		return;
	}

	// read the word until a whitespace character is found
	while (!file->eof) {
		DDP_FILL_READ_BUFFER_OR_RETURN_BREAK(file);

		// search for the next whitespace character
		char *whitespace_pos = strpbrk(read_buff_start_ptr(file), " \t\n\v\f\r");
		if (whitespace_pos) {
			size_t copy_amount = whitespace_pos - read_buff_start_ptr(file);
			if (whitespace_pos >= read_buff_end_ptr(file)) {
				copy_amount = read_buff_len(file);
			}

			// allocate space for the string + null terminator
			ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap + copy_amount + 1);
			// copy the string
			memcpy(ret->str + ret->cap, read_buff_start_ptr(file), copy_amount);
			ret->cap += copy_amount + 1; // + nullterminator
			ret->str[ret->cap - 1] = '\0';

			file->read_buffer_start += copy_amount; // consume up to but not including the whitespace
			return;
		} else {
			// no whitespace found, copy the whole buffer
			const size_t copy_amount = read_buff_len(file);

			// allocate space for the string (without '\0'!)
			ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap + copy_amount);
			// copy the string
			memcpy(ret->str + ret->cap, read_buff_start_ptr(file), copy_amount);
			ret->cap += copy_amount;

			file->read_buffer_start += copy_amount; // consume the whole buffer
		}
	}
	// EOF reached
	ret->str = ddp_reallocate(ret->str, ret->cap, ret->cap + 1);
	ret->str[ret->cap] = '\0';
	ret->cap++;
}

ddpbool Datei_Zuende(DateiRef datei) {
	DDP_MIGHT_ERROR;
	DDP_DBGLOG("Datei_Zuende: " DDP_INT_FMT ", " DDP_INT_FMT, datei->index, datei->id)

	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return true;
	}
	return file->eof;
}

void Datei_Schreibe_Zahl(DateiRef datei, ddpint i) {
	DDP_MIGHT_ERROR;

	// get the internal file and validate it
	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return;
	}
	DDP_ENSURE_WRITE_MODE(file);

	char buff[21]; // max_digits + sign + null terminator
	if (snprintf(buff, sizeof(buff), DDP_INT_FMT, i) < 0) {
		ddp_error("Fehler beim Schreiben der Zahl: snprintf failed", false);
		return;
	}
	write_to_buffer(file, buff, strlen(buff));
}

void Datei_Schreibe_Text(DateiRef datei, ddpstring *s) {
	DDP_MIGHT_ERROR;

	// get the internal file and validate it
	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return;
	}
	DDP_ENSURE_WRITE_MODE(file);

	write_to_buffer(file, s->str, s->cap - 1); // -1 to not write the null terminator
}

void Datei_Schreibe_Buchstabe(DateiRef datei, ddpchar c) {
	DDP_MIGHT_ERROR;

	// get the internal file and validate it
	InternalFile *file = get_internal_file(datei->index, datei->id);
	if (!file) {
		return;
	}
	DDP_ENSURE_WRITE_MODE(file);

	char buff[5]; // max utf-8 char length + null terminator
	int num_bytes = utf8_char_to_string(buff, c);
	if (num_bytes < 0) {
		ddp_error("Fehler beim Schreiben des Buchstabens: " DDP_CHAR_FMT " is not valid utf8", false, c);
		return;
	}
	write_to_buffer(file, buff, num_bytes);
}
