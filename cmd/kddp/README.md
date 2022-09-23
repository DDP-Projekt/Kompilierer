# kddp
Dieses Programm enthält Befehle um .ddp Dateien zu parsen, kompilieren und auszuführen.
## Befehle
|Befehlname|Befehlsyntax|Befehl beschreibung|Befehl optionen|Optionsbeschreibungen|
|------------|--------------|-------------------|---------------|------------------|
|hilfe|`hilfe <Befehl>`|Zeigt Nutzungsinformationen über den Befehl|-|-|
|kompiliere|`kompiliere <Eingabedatei> <Optionen>`|Kompiliert die gegebene .ddp Datei zu einer ausführbaren Datei|`-o <Ausgabepfad>`<br>`--wortreich`<br>`--nichts_loeschen`<br>`--gcc_optionen`<br>`--externe_gcc_optionen`|Optionaler Pfad der Ausgabedatei<br>Gibt wortreiche Informationen während des Befehls<br>Temporäre Dateien werden nicht gelöscht<br>Benutzerdefinierte Optionen, die gcc übergeben werden<br>Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden|
|parse|`parse <Eingabedatei> <Optionen>`|Parse die Eingabedatei zu einem Abstrakten Syntaxbaum|`-o <filepath>`|Optionaler Pfad der Ausgabedatei|
|version|`version <Optionen>`|Zeige informationen zu dieser DDP Version|`--wortreich`<br>`--go_build_info`|Zeige wortreiche Informationen<br>Zeige Go build Informationen|
|starte|`starte <Eingabedatei> <Optionen>`|Kompiliert und führt die gegebene .ddp Datei aus|`--wortreich`<br>`--gcc_optionen`<br>`--externe_gcc_optionen`|Gibt wortreiche Informationen während des Befehls<br>Benutzerdefinierte Optionen, die gcc übergeben werden<br>Benutzerdefinierte Optionen, die gcc für jede externe .c Datei übergeben werden|