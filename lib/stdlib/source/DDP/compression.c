#include "DDP/ddptypes.h"
#include "DDP/ddpos.h"
#include "DDP/error.h"
#include "DDP/ddptypes.h"
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <sys/stat.h>
#include <unistd.h>
#include <fcntl.h>

#define LIBARCHIVE_STATIC

#include <archive.h>
#include <archive_entry.h>

#define ARCHIVE_FILTER(packed) (packed & 0xF)
#define ARCHIVE_FORMAT(packed) ((packed & 0xFFFFF0) >> 4)

#define MAX_PATH 260
#define READ_BUFFER 8192

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

	//printf("Filter: %d, Format: 0x%x\n",  ARCHIVE_FILTER(type), ARCHIVE_FORMAT(type));	

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
	struct archive *disk = archive_read_disk_new();
	archive_read_disk_set_standard_lookup(disk);

	struct archive_entry *entry = archive_entry_new();
	int fd = open(path, O_RDONLY);
	if (fd < 0) {
		return 0;
	}

	archive_entry_copy_pathname(entry, path);
	archive_read_disk_entry_from_file(disk, entry, fd, NULL);
	archive_write_header(a, entry);

	char buff[READ_BUFFER];
	size_t bytes_read;
	while ((bytes_read = read(fd, buff, sizeof(buff))) > 0) {
		archive_write_data(a, buff, bytes_read);
	}

	archive_write_finish_entry(a);
	archive_read_free(disk);
	archive_entry_free(entry);
	return 1;
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

void copyData(struct archive *a, struct archive *disk) {
	int r;
	const void *buff;
	size_t size;
	int64_t offset;
	while ((r = archive_read_data_block(a, &buff, &size, &offset)) != ARCHIVE_EOF) {
		if (r != ARCHIVE_OK) {
			ddp_error("archive_read_data_block()", false, archive_error_string(disk));
			archive_read_free(a);
			archive_write_free(disk);
			return;
		}

		if (archive_write_data_block(disk, buff, size, offset) != ARCHIVE_OK) {
			ddp_error("archive_write_data_block()", false, archive_error_string(disk));
			archive_read_free(a);
			archive_write_free(disk);
			return;
		}
	}
}

// typ: | 16Bit format | 4 Bit filter | 
void Archiv_Aus_Dateien(ddpint typ, ddpstringlist *dateiPfade, ddpstring *arPfad) {
	DDP_MIGHT_ERROR;
	
	if (ddp_string_empty(arPfad)) {
		ddp_error("Invalid path", false);
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
    if (ddp_string_empty(ordnerPfad) || ddp_string_empty(arPfad)) {
        ddp_error("Leerer Pfad", false);
        return;
    }

    struct archive *a = createArchive(arPfad->str, (int)typ);
    if (!a) return;

    struct archive *disk = archive_read_disk_new();
    if (!disk) {
        ddp_error("Failed to create disk reader", false);
        archive_write_free(a);
        return;
    }

    archive_read_disk_set_standard_lookup(disk);

    if (archive_read_disk_open(disk, ordnerPfad->str) != ARCHIVE_OK) {
        ddp_error("Failed to open input directory: "DDP_STRING_FMT, false, archive_error_string(disk));
        archive_read_free(disk);
        archive_write_free(a);
        return;
    }

    struct archive_entry *entry;
    while (archive_read_next_header2(disk, entry = archive_entry_new()) == ARCHIVE_OK) {
        // Descend into directories as needed
        if (archive_entry_filetype(entry) == AE_IFDIR) {
            archive_read_disk_descend(disk);
        }

        if (archive_write_header(a, entry) != ARCHIVE_OK) {
            ddp_error("Failed to write header: "DDP_STRING_FMT, false, archive_error_string(a));
            archive_entry_free(entry);
            break;
        }

        const char *path = archive_entry_sourcepath(entry);
        if (path && archive_entry_filetype(entry) == AE_IFREG) {
            int fd = open(path, O_RDONLY);
            if (fd < 0) {
                ddp_error("Failed to open file: "DDP_STRING_FMT, true, path);
                archive_entry_free(entry);
                continue;
            }

            char buff[READ_BUFFER];
            ssize_t bytes_read;
            while ((bytes_read = read(fd, buff, sizeof(buff))) > 0) {
                if (archive_write_data(a, buff, bytes_read) != bytes_read) {
                    ddp_error("Failed to write file data: "DDP_STRING_FMT, false, archive_error_string(a));
                    break;
                }
            }
            close(fd);
        }

        archive_entry_free(entry);
        archive_write_finish_entry(a);
    }

    archive_read_free(disk);
    if (archive_write_close(a) != ARCHIVE_OK) {
        ddp_error("Failed to close archive: "DDP_STRING_FMT, false, archive_error_string(a));
    }
    archive_write_free(a);
}

void Archiv_Entpacken_Dateien_Pfad(ddpstringlist *dateiPfade, ddpstring *arPfad, ddpstring *ordnerPfad) {
	DDP_MIGHT_ERROR;
	if (ddp_string_empty(arPfad)) {
		ddp_error("Leerer Archivpfad gegeben", false);
		return;
	}

	struct archive *a = openArchive(arPfad->str);

	int flags = ARCHIVE_EXTRACT_TIME | ARCHIVE_EXTRACT_PERM | ARCHIVE_EXTRACT_ACL | ARCHIVE_EXTRACT_FFLAGS;

	struct archive *disk = archive_write_disk_new();
	archive_write_disk_set_options(disk, flags);
	archive_read_disk_set_standard_lookup(disk);

	int r;
	struct archive_entry *entry;
	while ((r = archive_read_next_header(a, &entry)) != ARCHIVE_EOF) {
		if (r != ARCHIVE_OK) {
			ddp_error("Failed to read header: "DDP_STRING_FMT, false, archive_error_string(a));
			archive_read_free(a);
			return;
		}

		// skip all entries not in dateiPfade
		if (dateiPfade->len != 0) {
			bool found = false;
			for (int i = 0; i < dateiPfade->len; i++) {
				if (!ddp_string_empty(&dateiPfade->arr[i]) && !strcmp(archive_entry_pathname(entry), dateiPfade->arr[i].str)) {
					found = true;
					break;
				}
			}

			if (!found) {
				continue;
			}
		}

		if (!ddp_string_empty(ordnerPfad)) {
			const char *original_path = archive_entry_pathname(entry);
			char full_path[MAX_PATH];
			snprintf(full_path, sizeof(full_path), "%s/%s", ordnerPfad->str, original_path);
			archive_entry_set_pathname(entry, full_path);
		}

		if (archive_write_header(disk, entry) != ARCHIVE_OK) {
			ddp_error("archive_write_header()", false, archive_error_string(disk));
			archive_read_free(a);
			archive_write_free(disk);
			return;
		}

		copyData(a, disk);
		
		if (archive_write_finish_entry(disk) != ARCHIVE_OK) {
			ddp_error("archive_write_finish_entry()", false, archive_error_string(disk));
			archive_read_free(a);
			archive_write_free(disk);
			return;
		}
	}

	archive_read_free(a);
  	archive_write_free(disk);
}

ddpint Archiv_Datei_Groesse(ddpstring *dateiPfad, ddpstring *arPfad) {
	DDP_MIGHT_ERROR;
	
	if (ddp_string_empty(arPfad) || ddp_string_empty(dateiPfad)) {
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

	if (ddp_string_empty(arPfad)) {
		return 0;
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
