# Changelog

Der Changelog von DDP. Sortiert nach Release.

**Legende**:
 - Added:    Etwas wurde hinzugefügt
 - Changed:  Etwas wurde geändert
 - Removed:  Etwas wurde entfernt
 - Fix:      Ein Bug wurde repariert.
 - Breaking: Die Änderung macht alte Programme kaputt

## In Entwicklung

- [Fix] crash bei Einbindungen von öffentlichen Kombinationen
- [Added]: zwischen operator hinzugefügt
- [Fix] typecheck crash bei Typumwandlungen zu invaliden Typen

## v0.1.0-alpha

- [Added]: Beliebige utf-8 Symbole sind jetzt in Aliasen erlaubt
- [Breaking] `Boolean` zu `Wahrheitswert` umbennant
- [Fix] `kddp update` ignoriert die -jetzt flag nicht mehr
- [Fix] `kddp update` updated jetzt auch den Duden
- [Added] Datei_Kopieren Funktion zu Duden/Dateisystem 
- [Fix] Bessere Fehlermeldungen [#28](https://github.com/DDP-Projekt/Kompilierer/pull/28)
- [Added] Datei-info Funktionen zu Duden/Dateisystem 
- [Fix] Double-Free Fehler in externen Funktionen
- [Breaking] `von...bis` wurde zu `im Bereich von...bis` umbenannt
- [Added] Syntax wie in Deklarationen für boolesche Rückgabewerte
- [Added] `bis...zum` und `ab...dem`
- [Fix] Alias Deklarationen
- [Breaking] Namenskonvention im Duden
- [Changed] Verbesserungen am Visitor interface (für den LS)
- [Fix] Typos
- [Breaking] Duden/Extremwerte nach Duden/Zahlen verschoben
- [Added] Duden/Zahlen
- [Added] Mehr Duden/Texte und Duden/Zeichen tests
- [Fix] Duden/Texte edge-cases
- [Fix] Verschiedene Crashes
- [Added] Strukturen [#5](https://github.com/DDP-Projekt/Kompilierer/issues/5)

## v0.0.1-alpha
- Erster Release von DDP