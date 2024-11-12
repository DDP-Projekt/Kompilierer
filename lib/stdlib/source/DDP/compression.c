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

void Archiv_Aus_Ordner(ddpstring *ordnerPfad, ddpstring *arPfad) {

}

void Archiv_Aus_Datei(ddpstring *dateiPfad, ddpstring *arPfad) {

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