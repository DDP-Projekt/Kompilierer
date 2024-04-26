# Changelog

Der Changelog von DDP. Sortiert nach Release.

**Legende**:
 - Added:    Etwas wurde hinzugefügt
 - Changed:  Etwas wurde geändert
 - Removed:  Etwas wurde entfernt
 - Fix:      Ein Bug wurde repariert.
 - Breaking: Die Änderung macht alte Programme kaputt

## In Entwicklung

- [Fix] Der zwischen Operator crasht nun nicht mehr bei Kommazahl/Zahl kombinationen
- [Fix] Die Reihenfolge der 2 letzten Argumente beim zwischen Operator ist jetzt egal
- [Added] Optionale Optimierungsstufe 2 optimiert das Kopieren von komplexeren Datentypen
- [Added] Befehlszeilenargument "-O/--optimierungs-stufe" um die Optimierungsstufe zu setzen
- [Changed] Befehlszeilenargumente benutzen nun "-" anstatt "_" (z.B. --nichts-loeschen anstatt --nichts_loeschen). "_" kann allerdings immernoch benutzt werden
- [Added] der Standardwert Operator gibt den Standardwert eines Typen zurück
- [Breaking] Der Größe Operator nimmt nun einen Typ als Operanden
- [Fix] Bug beim Vergleichen von Kombinationen
- [Added] Folgende Duden funktionen wurden hinzugefügt:
  - Text- und Zeichenvergleichsfunktionen
  - Elementweise Operationen für Listen
  - Um Buchstaben und Text Listen zu einem Text aneinanderhängen
  - Summe und Produkt für numerische Listen
  - Wrapper Funktionen für ' ', '\n', '\r', '\t', '\\', '"' und '\'" in Duden/Zeichen 
  - Natürlicher_Logarithmus, Gaußsche_Fehlerfunktion und Fakultät
  - Kosekans, Sekans, Kotangens, Areahyperbelsinus, Areahyperbelkosinus, Areahyperbeltangens, Versinus und Koversinus
  - einige Regex funktionen im neuen Duden/Regex Modul 
  - Erste_N_Elemente_X, Letzten_N_Elemente_X und Spiegeln_X,
  - SHA-256 und SHA-512 zum neuen Duden/Kryptographie Modul
  - Bogenmaß_Zu_Grad, Grad_Zu_Bogenmaß, atan2 und Kehrwert
  - Aufsteigende_Zahlen, Absteigende_Zahlen, Linspace und Logspace
  - Hamming_Distanz und Levenshtein_Distanz
  - Max3, Min3, Max3_Kommazahl und Min3_Kommazahl
- [Fix] Externe Funktionsnamen in random.c
- [Fix] Bei allen geklammerten Argumenten werden Fehler jetzt korrekt gemeldet

## v0.2.0-alpha

- [Fix] Bei geklammerten Referenz Argumenten werden Fehler jetzt korrekt gemeldet
- [Breaking] Duden/Fehlerbehandlung wird nun überall im Duden benutzt
- [Fix] Erstelle_Ordner gibt keinen Fehler mehr zurück, wenn einer der Ordner bereits existiert
- [Added] Duden/Fehlerbehandlung
- [Fix] crash bei Einbindungen von öffentlichen Kombinationen
- [Added]: zwischen operator hinzugefügt
- [Fix] typecheck crash bei Typumwandlungen zu invaliden Typen

## v0.1.0-alpha

- [Added] Beliebige utf-8 Symbole sind jetzt in Aliasen erlaubt
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
