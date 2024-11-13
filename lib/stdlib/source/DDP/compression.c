#include "DDP/ddptypes.h"
#include "DDP/ddpos.h"
#include <archive.h>
#include <archive_entry.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dirent.h>
#include <sys/stat.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>

// typ: | 16Bit format | 4 Bit filter | 
void Archiv_Aus_Datei(ddpint typ, ddpstring *dateiPfad, ddpstring *arPfad) {
	printf("Adding file %s to archive at path %s\n", dateiPfad->str, arPfad->str);

    // Create a new archive object for writing
    struct archive *a = archive_write_new();
    if (!a) {
        fprintf(stderr, "Failed to create archive\n");
        return;
    }

    // Extract filter bits and set the compression filter
    if (archive_write_add_filter(a, typ & 0xF) != ARCHIVE_OK) {
        fprintf(stderr, "Failed to set compression algorithm: %s\n", archive_error_string(a));
        archive_write_free(a);
        return;
    }

    // Extract format bits and set the file format
    if (archive_write_set_format(a, (typ & 0xFFFFF0) >> 4) != ARCHIVE_OK) {
        fprintf(stderr, "Failed to set archive format: %s\n", archive_error_string(a));
        archive_write_free(a);
        return;
    }

    if (archive_write_open_filename(a, arPfad->str) != ARCHIVE_OK) {
        fprintf(stderr, "Failed to open output file: %s\n", archive_error_string(a));
        archive_write_free(a);
        return;
    }

	struct stat st;
	if (stat(dateiPfad->str, &st) < 0) {
        fprintf(stderr, "Failed to stat input file: %s\n", strerror(errno));
        archive_write_free(a);
        return;
    }

    // Create a new entry for the file
    struct archive_entry *entry = archive_entry_new();
    archive_entry_set_pathname(entry, dateiPfad->str);
    archive_entry_copy_stat(entry, &st);
	archive_entry_set_perm(entry, 0644);

    // Write the header for the entry
    if (archive_write_header(a, entry) != ARCHIVE_OK) {
        fprintf(stderr, "Failed to write header: %s\n", archive_error_string(a));
        archive_entry_free(entry);
        archive_write_free(a);
        return;
    }

    // Open the input file
    FILE *file = fopen(dateiPfad->str, "rb");
    if (!file) {
        fprintf(stderr, "Failed to open input file: %s\n", dateiPfad->str);
        archive_write_free(a);
        return;
    }

    char buffer[8192];
    size_t bytes_read;
    // Write the file data to the archive entry
    while ((bytes_read = fread(buffer, 1, sizeof(buffer), file)) > 0) {
        if (archive_write_data(a, buffer, bytes_read) != bytes_read) {
            fprintf(stderr, "Failed to write data: %s\n", archive_error_string(a));
            fclose(file);
            archive_entry_free(entry);
            archive_write_free(a);
            return;
        }
    }

    // Clean up
    fclose(file);
    archive_entry_free(entry);

    // Close the archive
    if (archive_write_close(a) != ARCHIVE_OK) {
        fprintf(stderr, "Failed to close archive: %s\n", archive_error_string(a));
        archive_write_free(a);
        return;
    }

    // Free the archive object
    archive_write_free(a);
}

void Archiv_Aus_Ordner(ddpint typ, ddpstring *ordnerPfad, ddpstring *arPfad) {
	
}

void Archiv_Entpacken_Ordner(ddpstring *arPfad, ddpstring *ordnerPfad) {

}

void Archiv_Entpacken_Datei(ddpstring *dateiPfad, ddpstring *arPfad, ddpstring *pfadZu) {

}

void Archiv_Ordner_Hinzufuegen(ddpstring *ordnerPfad, ddpstring *arPfad) {

}

void Archiv_Datei_Hinzufuegen(ddpstring *dateiPfad, ddpstring *arPfad) {

}

void Archiv_Ordner_Entfernen(ddpstring *ordnerPfad, ddpstring *arPfad) {

}

void Archiv_Dateien_Entfernen(ddpstringlist *dateiPfade, ddpstring *arPfad) {

}

ddpint Archiv_Datei_Groesse_Komp(ddpstring *dateiPfad, ddpstring *arPfad) {
	return 0;
}

ddpint Archiv_Datei_Groesse_Unkomp(ddpstring *dateiPfad, ddpstring *arPfad) {
	return 0;
}

ddpint Archiv_Anzahl_Dateien(ddpstring *arPfad) {
	return 0;
}
