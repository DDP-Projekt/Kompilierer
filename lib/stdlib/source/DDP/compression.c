#include "DDP/ddptypes.h"
#include "DDP/ddpos.h"
#include "DDP/error.h"
#include <archive.h>
#include <archive_entry.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <sys/stat.h>
#include <unistd.h>
#include <fcntl.h>

#define ARCHIVE_FILTER(packed) (packed & 0xF)
#define ARCHIVE_FORMAT(packed) ((packed & 0xFFFFF0) >> 4)

// creates an archive at the given path with filetype and compression algorithm packed into the type.
// see ARCHIVE_FILTER(packed) and ARCHIVE_FORMAT(packed).
// returns the archive object if successful and NULL (0) if not.
struct archive* createArchive(const char *path, int type) {
	// Create a new archive object for writing
	struct archive *a = archive_write_new();
	if (!a) {
		ddp_error("Failed to create archive", false);
		return NULL;
	}

	// Extract filter bits and set the compression filter
	if (archive_write_add_filter(a, ARCHIVE_FILTER(type)) != ARCHIVE_OK) {
		ddp_error("Failed to set compression algorithm: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_write_free(a);
		return NULL;
	}

	// Extract format bits and set the file format
	if (archive_write_set_format(a, ARCHIVE_FORMAT(type)) != ARCHIVE_OK) {
		ddp_error("Failed to set archive format: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_write_free(a);
		return NULL;
	}

	switch (ARCHIVE_FORMAT(type)) {
		case ARCHIVE_FORMAT_7ZIP:
			archive_write_set_options(a, "7zip:compression=lzma2");
			break;
		case ARCHIVE_FORMAT_ZIP:
			archive_write_set_options(a, "zip:compression=lzma2");
			break;
	}

	if (archive_write_open_filename(a, path) != ARCHIVE_OK) {
		ddp_error("Failed to open output file: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_write_free(a);
		return NULL;
	}

	return a;
}

// create an entry in the given archive with the content of the file at the given path.
// returns true (1) if successful and false (0) if not.
int createEntry(struct archive *a, const char *path) {
	struct stat st;
	if (stat(path, &st) < 0) {
		ddp_error("Failed to stat input file "DDP_STRING_FMT" :", true, path);
		archive_write_free(a);
		return 0;
	}

	// Create a new entry for the file
	struct archive_entry *entry = archive_entry_new();
	if (!entry) {
		ddp_error("Failed to create entry", false);
		archive_write_free(a);
		return 0;
	}
	
	archive_entry_set_pathname(entry, path);
	archive_entry_copy_stat(entry, &st);
	archive_entry_set_perm(entry, 0644);

	// Write the header for the entry
	if (archive_write_header(a, entry) != ARCHIVE_OK) {
		ddp_error("Failed to write header: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_entry_free(entry);
		archive_write_free(a);
		return 0;
	}

	// Open the input file
	FILE *file = fopen(path, "rb");
	if (!file) {
		ddp_error("Failed to open input file: "DDP_STRING_FMT, false, path);
		archive_write_free(a);
		return 0;
	}

	char buffer[8192];
	size_t bytes_read;
	// Write the file data to the archive entry
	while ((bytes_read = fread(buffer, 1, sizeof(buffer), file)) > 0) {
		if (archive_write_data(a, buffer, bytes_read) != bytes_read) {
			ddp_error("Failed to write data: "DDP_STRING_FMT, false, archive_error_string(a));
			fclose(file);
			archive_entry_free(entry);
			archive_write_free(a);
			return 0;
		}
	}

	// Clean up
	fclose(file);
	archive_entry_free(entry);
	return 1;
}

// walks a directory and creates a new entry for each file
void compressDir(struct archive *a, const char *path) {
	DIR* dir;
	struct dirent *ent;
	struct stat states;

	dir = opendir(path);
	if (!dir) {
		ddp_error("Couldn't open directory "DDP_STRING_FMT": ", true, path);
		return;
	}

	while ((ent=readdir(dir)) != NULL) {
		if (!strcmp(".", ent->d_name) || !strcmp("..", ent->d_name)) {
			continue;
		}

		char localname[255];
		strcpy(localname, path);
		strcat(localname, "/");
		strcat(localname, ent->d_name);

		if (stat(localname, &states) < 0) {
			ddp_error("Failed to stat input file "DDP_STRING_FMT": ", true, ent->d_name);
			return;
		}
		
		if (S_ISDIR(states.st_mode)) {
			compressDir(a, localname);
		}
		else {
			if (!createEntry(a, localname)) {
				closedir(dir);
				return;
			}
		}
	}

    closedir(dir);
}

struct archive* openArchive(const char *path) {
	struct archive *a = archive_read_new();
	if (archive_read_support_filter_all(a) != ARCHIVE_OK) {
		ddp_error("Failed to determine archive compression algorithm: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_read_free(a);
		return NULL;
	}

	if (archive_read_support_format_all(a) != ARCHIVE_OK) {
		ddp_error("Failed to determine archive file format: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_read_free(a);
		return NULL;
	}

	if (archive_read_open_filename(a, path, 10240) != ARCHIVE_OK) {
		ddp_error("Failed to open archive: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_read_free(a);
		return NULL;
	}

	return a;
}

// typ: | 16Bit format | 4 Bit filter | 
void Archiv_Aus_Dateien(ddpint typ, ddpstringlist *dateiPfade, ddpstring *arPfad) {
	DDP_MIGHT_ERROR;
	if (dateiPfade == NULL || arPfad == NULL) {
		ddp_error("Invalid path", false);
		return;
	}

	if (dateiPfade->len == 0) {
		ddp_error("No paths given", false);
		return;
	}

	struct archive *a = createArchive(arPfad->str, (int)typ);
	if (!a) return;

	for (int i = 0; i < dateiPfade->len; i++) {
		if (dateiPfade->arr[i].str == NULL) {
			ddp_error("Invalid path", false);
			archive_write_free(a);
			return;
		}

		if (!createEntry(a, dateiPfade->arr[i].str)) {
			return;
		}
	}
	
	// Close the archive
	if (archive_write_close(a) != ARCHIVE_OK) {
		ddp_error("Failed to close archive: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_write_free(a);
		return;
	}

	// Free the archive object
	archive_write_free(a);
}

void Archiv_Aus_Ordner(ddpint typ, ddpstring *ordnerPfad, ddpstring *arPfad) {
	DDP_MIGHT_ERROR;
	if (ordnerPfad == NULL || arPfad == NULL) {
		ddp_error("Invalid path", false);
		return;
	}

	struct archive *a = createArchive(arPfad->str, (int)typ);
	if (!a) return;

	compressDir(a, ordnerPfad->str);

	// Close the archive
	if (archive_write_close(a) != ARCHIVE_OK) {
		ddp_error("Failed to close archive: "DDP_STRING_FMT, false, archive_error_string(a));
		archive_write_free(a);
		return;
	}

	// Free the archive object
	archive_write_free(a);
}

void Archiv_Entpacken_Ordner(ddpstring *arPfad, ddpstring *ordnerPfad) {

}

void Archiv_Entpacken_Dateien(ddpstringlist *dateiPfade, ddpstring *arPfad, ddpstring *pfadZu) {

}

void Archiv_Ordner_Hinzufuegen(ddpstring *ordnerPfad, ddpstring *arPfad) {

}

void Archiv_Dateien_Hinzufuegen(ddpstring *dateiPfad, ddpstring *arPfad) {

}

void Archiv_Ordner_Entfernen(ddpstring *ordnerPfad, ddpstring *arPfad) {

}

void Archiv_Dateien_Entfernen(ddpstringlist *dateiPfade, ddpstring *arPfad) {

}

ddpint Archiv_Datei_Groesse(ddpstring *dateiPfad, ddpstring *arPfad) {
	DDP_MIGHT_ERROR;
	if (!arPfad || !dateiPfad) {
		return -1;
	}
	
	struct archive *a = openArchive(arPfad->str);
	if (!a) return -1;

	// find entry
	struct archive_entry *e;
	while (archive_read_next_header(a, &e) == ARCHIVE_OK) {
		if (!strcmp(dateiPfad->str, archive_entry_pathname(e))) {
			break;
		}
	}

	la_ssize_t size = archive_entry_size(e);

	if (archive_read_free(a) != ARCHIVE_OK) {
		ddp_error("Failed to close archive: "DDP_STRING_FMT, false, archive_error_string(a));
		return -1;
	}

	return (ddpint)size;
}

ddpint Archiv_Anzahl_Dateien(ddpstring *arPfad) {
	DDP_MIGHT_ERROR;
	int count = -1;

	if (!arPfad) {
		return -1;
	}
	
	struct archive *a = openArchive(arPfad->str);
	if (!a) return -1;

	// iterate over every file
	struct archive_entry *e;
	while (archive_read_next_header(a, &e) == ARCHIVE_OK);

	count = archive_file_count(a);

	if (archive_read_free(a) != ARCHIVE_OK) {
		ddp_error("Failed to close archive: "DDP_STRING_FMT, false, archive_error_string(a));
		return -1;
	}

	return (ddpint)count;
}
