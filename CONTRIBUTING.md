# Zu DDP beitragen

DDP ist und wird immer Open-Source bleiben und alle Beiträge sind willkommen! Jeder kann durch Github Pull-Requests zur `dev` branch zu dem Projekt beitragen.

# Den DDP Kompilierer bauen

## Vorrausetzungen

Der DDP Kompilierer ist in Go geschrieben. Daher wird eine Go Version von 1.22.2 oder höher benötigt. Diese kann hier gedownloaded werden: https://go.dev/dl/.<br>
Hinweis: Manche Packet-Manager installieren nicht die korrekte Version von Go. Die Go Version kann mit dem Befehl `go version` überprüft werden.

Um den DDP Kompilierer zu bauen, muss LLVM 12.0 installiert sein. Bei vielen Linux Distributionen geht das ganz einfach mit `sudo apt install llvm-12` bzw. `sudo apt install llvm-12-dev`.
Unter Windows muss LLVM lokal selbst gebaut werden (das geht auch bei Linux, wird aber nicht empfohlen, da es einige Stunden dauern kann).
Allerdings gibt es für die von uns genutzen MinGW Versionen bereits [vorkompilierte LLVM Bibliotheken](https://github.com/DDP-Projekt/Kompilierer/releases/tag/llvm-binaries).
Bei diesen findet sich auch eine Linux version.

Wer LLVM auf Linux trotzdem selber bauen möchte, kann zum *LLVM bauen* Teil springen.

Um das Makefile auf Windows auszuführen müssen folgende Programme installiert und im PATH sein:

- [Go](https://go.dev/dl/) (mindestens Version 1.22.2)
- [mingw64](https://winlibs.com/) (Kann auch über chocolatey installiert werden.
    Version 12.2.0 mit der msvcrt oder ucrt runtime wurde getestet und funktioniert, andere Versionen werden vielleicht nicht funktionieren)
- make (kann man mit chocolatey bekommen oder durch mingw32-make, was mit mingw64 dazu kommt)
- [git](https://git-scm.com/download/win) (git-bash wird benutzt um die Makefiles auszuführen)
- [CMake](https://cmake.org/download/) (Version 3.13.4 oder höher)

### Hinweis für Windows

Das endgültige DDP-Build ist eng mit der verwendeten mingw64-Version gekoppelt.
Da DDP GCC als Linker und für die C-Standardbibliothek verwendet,
benötigt das endgültige DDP-Build dieselbe mingw64-Version mit der es gebaut wurde, um DDP-Programme zu kompilieren.
Wenn man DDP auf einem beliebigen Computer einrichten möchte sollte man daran denken.

## Unter Linux

Nachdem LLVM installiert ist, muss einfach der Befehl `make` im Stammverzeichnis des Repositoriums ausgeführt werden.

Um die Tests auszuführen, wird `make test` verwendet.
`make help` zeigt eine kurze Erklärung aller make Ziele.

Um LLVM aus dem Submodul llvm-project zu erstellen, wird `make llvm` verwendet.
Falls LLVM aus dem Submodul gebaut wurde, wird make dieses LLVM-Build anstelle des Globalen verwenden.

Es ist empfohlen die j Option von make zu benutzen um mehr als einen Thread zu benutzen. Beispiel: `make -j8`.

## Unter Windows

Nachdem LLVM installiert ist (siehe *LLVM bauen* oder *Vorkompiliertes LLVM*), müssen die gleichen Schritte wie unter Linux befolgt werden, wobei allerdings git-bash benutzt wird um die Befehle auszuführen (sonst funktionieren einige Unix-Befehle nicht).

### Wichtig

GNUMake wird nicht funktionieren. Wenn gnu make installiert ist, hat es unter Windows den Namen *make*, daher sollte stattdessen mingw32-make verwendet werden, um die Makefiles ausführen.

# Vorkompiliertes LLVM

Für bestimmte MinGW Versionen gibt es vorkompilierte (LLVM Bibliotheken)[https://github.com/DDP-Projekt/Kompilierer/releases/tag/llvm-binaries].
Diese können einfach heruntergeladen und in den Ordner `llvm_build` entpackt werden.

Die verwendete MinGW muss mit der LLVM Version übereinstimmen, da es sonst zu Fehlern kommen wird.

## Komplettes Windows Beispiel mit vorkompiliertem LLVM und MinGW via Chocolatey

Alle Befehle wurden im Root des Repositoriums ausgeführt.

```bash
$ choco install mingw --version 12.2.0
# download and extract llvm binaries
$ curl -L -o ./llvm_build.tar.gz https://github.com/DDP-Projekt/Kompilierer/releases/download/llvm-binaries/llvm_build-mingw-12.2.0-x86_64-ucrt-posix-seh.tar.gz
$ mkdir -p ./llvm_build/
$ tar -xzf ./llvm_build.tar.gz -C ./ --force-local
$ rm ./llvm_build.tar.gz
# build the compiler
$ make -j8
# run the tests
$ make test-complete -j8
```

# LLVM bauen

## Voraussetzungen

- [Python3](https://www.python.org/downloads/)

Unter Windows sind die oben genannten Voraussetzungen ebenfalls erforderlich, und alle folgenden Befehle müssen in git-bash und nicht in cmd oder Powershell ausgeführt werden.

Nachdem alle Voraussetzungen installiert sind, muss ein Terminal (auf Windows git-bash) im Stammverzeichnis des Repositoriums geöffnet und der Befehl `make llvm -j<Anzahl Threads>` ausgeführt werden.
Dadurch sollte das Untermodul llvm-project heruntergeladen und in llvm_build gebaut werden.
Der Download ist ungefähr 1 GB groß, daher kann es eine Weile dauern.
Das Bauen von LLVM dauert auf einem einzelnen Thread ebenfalls 2-3 Stunden, kann aber im Hintergrund ausgeführt werden, da es nicht viele Ressourcen beansprucht.
Um das ganze zu beschleunigen sollte die j Option von make benutzt werden um mehr als einen Thread zu benutzen.
Wenn du deinen PC nebenbei noch verwenden willst sollte ein Thread weniger als vorhandene CPU-Kerne benutzt werden.

# Vollständiges Beispiel unter Windows

Alle befehle wurden im Root des Repositoriums ausgeführt.

```
$ make llvm -j8
$ make -j8
```

# Am Kompilierer arbeiten

## Den Kompilierer Debuggen

Da der Kompilierer von den C-Bindings von LLVM abhängt kann er nicht wie ein normales Go-Programm debugget werden.
Vorher müssen einige Umgebungsvariablen gesetzt werden. Um diesen Prozess zu vereinfachen kann das Shell-Skript `open_for_debug.sh` verwendet werden.
Es setzt die enstprechenden Umgebungsvariablen und öffnet dann VSCode.
Für andere IDEs kann das Skript angepasst werden.