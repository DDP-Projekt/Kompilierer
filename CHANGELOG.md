# Changelog

Der Changelog von DDP. Sortiert nach Release.

**Legende**:
 - Added:    Etwas wurde hinzugefügt
 - Changed:  Etwas wurde geändert
 - Removed:  Etwas wurde entfernt
 - Fix:      Ein Bug wurde repariert.
 - Breaking: Die Änderung macht alte Programme kaputt

## In Entwicklung

- [Added] Duden/Uri
- [Added] Funktionen in Duden/Texte: Text_Worte und Spalten_Spaltmenge_Text
- [Added] Duden/TextIterator um effizient über Texte zu iterieren
- [Fix] Der "als" Operator kann nun für verschiedene Rückgabetypen überladen werden
- [Fix] Man kann eine Variable, die eine andere überschreibt jetzt mit dieser initialisieren
- [Added] Man kann jetzt (auch rekursiv) alle Module aus einem Ordner einbinden
- [Fix] Vorwärts Deklarationen geben nun keinen "undefined reference" Fehler mehr, wenn man sie einbindet

## v0.5.0-alpha

- [Added] Vorwärts Deklarationen
- [Added] _Ref Versionen für einige Duden/Listen und Duden/Texte Funktionen
- [Changed] Iterierenden Schleifen über Texte haben nun eine Zeitkomplexität von O(n) (anstatt O(n^2))
- [Fix] utf8 Texte
- [Fix] Aliase mit Referenz Parametern werden nun in mehr Fällen bevorzugt
- [Fix] Der Kompilierer crashet nicht mehr wenn indirekt importierte Symbole in eingebundenen Kombinations Aliasen verwendet werden
- [Changed] Duden/Zeichen und Duden/Texte um Konflikte mit dem neuen Variablen Typ zu vermeiden:
    - Buchstabe_Ist_Zahl -> Buchstabe_Ist_Ziffer (Alias ebenfalls angepasst)
    - Text_Ist_Zahl: Alias angepasst
- [Added] Duden/Dateisystem Datei_Lies_Alles
- [Added] "Variable" als Typ, der zur Laufzeit jeder beliebige andere Typ sein kann
- [Added] ... als Platzhalter
- [Fix] Bug im Parser, der rekursiv allen Arbeitsspeicher verbraucht
- [Added] Operatoren-Überladung
- [Changed] Die Typen, die von Funktionen, Variablen und anderen Typen benutzt werden müssen jetzt nicht mehr extra eingebunden werden
- [Added] Typ-Aliase und Typ-Definitionen

## v0.4.0-alpha

- [Added] Aliasnegationen
- [Added] `entweder ..., oder` Operator
- [Changed] "ist" nach Vergleichen ist jetzt Optional, falls davor bereits ein "ist" steht
- [Added] Syntax um Variablen und Funktionen als "extern sichtbar" zu markieren, und somit name-mangling für diese auszuschalten
- [Fix] Linker Fehler bei mehreren öffentlichen Symbolen mit demselben Namen
- [Added] Falls Operator. Funktioniert so wie der Ternary Conditional Operator (?:) in anderen Sprachen. 
- [Added] in Duden/Dateisystem:
    - Datei Kombination
    - Datei_Oeffnen
    - Datei_Oeffnen_Lesen
    - Datei_Oeffnen_Schreiben
    - Datei_Oeffnen_Lesen_Schreiben
    - Datei_Oeffnen_Rückgabe
    - Datei_Oeffnen_Lesen_Rückgabe
    - Datei_Oeffnen_Schreiben_Rückgabe
    - Datei_Oeffnen_Lesen_Schreiben_Rückgabe
    - Datei_Schliessen
    - Datei_Zuende
    - Datei_Nicht_Zuende
    - Datei_Lies_N_Zeichen
    - Datei_Lies_Zeile
    - Datei_Lies_N_Zeilen
    - Datei_Lies_Wort
    - Datei_Lies_N_Worte
    - Datei_Lies_Zahl
    - Datei_Lies_N_Zahlen
    - Datei_Lies_Kommazahl
    - Datei_Lies_N_Kommazahlen
    - Datei_Schreibe_Zahl
    - Datei_Schreibe_Text
    - Datei_Schreibe_Kommazahl
    - Datei_Schreibe_Buchstabe
    - Datei_Schreibe_Wahrheitswert
    - Datei_Schreibe_Zeile_Zahl
    - Datei_Schreibe_Zeile_Text
    - Datei_Schreibe_Zeile_Kommazahl
    - Datei_Schreibe_Zeile_Buchstabe
    - Datei_Schreibe_Zeile_Wahrheitswert

- [Added] Runden in Mathe/Duden hinzugefügt
- [Fix] Funktionsparameter können nun nicht mehr Funktions- oder Kombinationsdeklarationen überschreiben

## v0.3.0-alpha

- [Breaking] Folgende Duden Funktionen wurden umbennant:
    - Pfade:
        - Ist_Absolut -> UNIX_Ist_Absolut 
        - Säubern -> UNIX_Säubern 
        - Verbinden -> UNIX_Verbinden 
        - Ordner_Pfad -> UNIX_Ordnerpfad 
        - Basis_Pfad -> UNIX_Basisname  
        - Erweiterung -> UNIX_Erweiterung 
        - Datei_Name -> UNIX_Dateiname 
    - Mathe: 
        - sin -> Sinus
        - cos -> Kosinus
        - tan -> tangens
        - asin -> Arkussinus
        - acos -> Arkuskosinus
        - atan -> Arkustangens
        - sinh -> Hyperbelsinus
        - cosh -> Hyperbelkosinus
        - tanh -> Hyperbeltangens 
- [Added] Folgende Duden Funktionen wurden hinzugefügt (siehe die [Bedienungsanleitung](https://ddp.le0n.dev/Bedienungsanleitung/de/Programmierung/Standardbibliothek/) für eine Beschreibung):
    - Pfade: 
        - UNIX_Vollständiger_Pfad  
        - Windows_Ist_Absolut 
        - Windows_Saeubern 
        - Windows_Pfad_Verbinden 
        - Windows_Laufwerkbuchstabe 
        - Windows_Vollständiger_Pfad 
        - Windows_Ordnerpfad 
        - Windows_Basisname 
        - Windows_Dateiname 
        - Windows_Erweiterung 
        - Wrapper Funktionen für Windows/Linux Funktionen
    - Laufzeit: 
        - Arbeitsverzeichnis 
    - Zeichen: 
        - Leerzeichen
        - Neue_Zeile
        - Wagenrücklauf
        - Tabulator
        - Rückstrich
        - Anführungszeichen
        - Apostroph
        - ASCII_Größer
        - ASCII_Kleiner,
        - Ist_Lateinischer_Buchstabe
        - Ist_Lateinischer_Buchstabe_Oder_Zahl 
    - Regex: 
        - Ist_Regex 
        - Regex_Erster_Treffer 
        - Regex_N_Treffer 
        - Regex_Alle_Treffer 
        - Regex_Erster_Treffer_Ersetzen 
        - Regex_Alle_Treffer_Ersetzen 
        - Regex_Spalten 
    - Kryptographie: 
        - SHA_256
        - SHA_512 
    - Listen: 
        - Aufsteigende_Zahlen
        - Absteigende_Zahlen, Linspace
        - Logspace, Erste_N_Elemente_X
        - Letzten_N_Elemente_X
        - Spiegeln_X
        - Summe_Zahlen_Liste
        - Produkt_Zahlen_Liste
        - Summe_Kommazahlen_Liste
        - Produkt_Kommazahlen_Liste
        - Verketten_Text_Liste 
        - Elementweise_Summe_Zahl/Kommazahl
        - Elementweise_Differenz_Zahl/Kommazahl
        - Elementweise_Produkt_Zahl/Kommazahl
        - Elementweise_Quotient_Zahl/Kommazahl
        - Elementweise_Verketten_Text 
    - Mathe: 
        - Min3
        - Min3_Kommazahl
        - Max3
        - Max3_Kommazahl
        - Bogenmaß_Zu_Grad
        - Winkel, Kehrwert_Zahl
        - Kehrwert_Kommazahl
        - Natürlicher_Logarithmus
        - Gaußsche_Fehlerfunktion
        - Fakultät *(Danke Franz Müller!)*
    - Texte: 
        - Hamming_Distanz
        - Levenshtein_Distanz
        - Vergleiche_Text 
        - Text_Anzahl_Buchstabe 
        - Text_Anzahl_Text 
        - Text_Anzahl_Text_Nicht_Überlappend 
        - Buchstaben_Text_BuchstabenListe 
        - Buchstaben_Text_TextListe 
        - Text_Index_Von_Text 
        - Spalte_Text 
        - Erster_Buchstabe 
        - Nter_Buchstabe 
        - Letzter_Buchstabe 
        - Finde_Subtext 
- [Fix] Der zwischen Operator crasht nun nicht mehr bei Kommazahl/Zahl kombinationen
- [Fix] Die Reihenfolge der 2 letzten Argumente beim zwischen Operator ist jetzt egal
- [Added] Optionale Optimierungsstufe 2 optimiert das Kopieren von komplexeren Datentypen
- [Added] Befehlszeilenargument "-O/--optimierungs-stufe" um die Optimierungsstufe zu setzen
- [Changed] Befehlszeilenargumente benutzen nun "-" anstatt "_" (z.B. `--nichts-loeschen` anstatt `--nichts_loeschen`). "_" kann allerdings immernoch benutzt werden
- [Added] der Standardwert Operator gibt den Standardwert eines Typen zurück
- [Breaking] Der Größe Operator nimmt nun einen Typ als Operanden
- [Fix] Bug beim Vergleichen von Kombinationen
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
