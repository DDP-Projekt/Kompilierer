# Zu DDP beitragen
DDP ist und wird immer Open-Source bleiben und alle Beiträge sind willkommen! Sie können zu diesem Projekt beitragen indem Sie Pull-Requests erstellen.

# Den DDP Kompilierer bauen
## Vorrausetzungen
Der DDP Kompilierer ist in Go geschrieben. Daher müssen Sie eine Golang version von 1.18 oder höher haben. Diese können Sie hier downloaden: https://go.dev/dl/.<br>
Hinweis: Manche Packet-Manager installieren nicht die korrekte version von Go. Bitte überprüfen Sie die Go version mit dem Befehl: `go version`.

Um den DDP Kompilierer zu bauen, müssen Sie LLVM 12.0 installiert haben. Bei vielen Linux Distributionen geht das ganz einfach mit `sudo apt install llvm-12`.
Unter Windows müssen Sie LLVM selber bauen (das können Sie auch bei Linux, wird aber nicht empfohlen, da es einige Stunden dauern kann).

Wenn Sie trotzdem LLVM auf linux bauen möchten, springen sie zum *LLVM bauen* Teil.

Um den Makefile auf Windows ausführen zu können müssen Sie folgende Programme installiert und in ihrem PATH haben:

- [Go](https://go.dev/dl/) (mindestens version 1.18)
- [mingw64](https://sourceforge.net/projects/mingw-w64/files/Toolchains%20targetting%20Win64/Personal%20Builds/mingw-builds/8.1.0/threads-posix/seh/x86_64-8.1.0-release-posix-seh-rt_v6-rev0.7z/download) (version 8.1.0 wurde getestet und funktioniert, andere versionen werden vielleicht nicht funktionieren)
- make (kann man mit chocolatey bekommen oder durch mingw32-make, was mit mingw64 dazu kommt)
- [git](https://git-scm.com/download/win) (Sie werden git-bash benutzen um die makefiles auszuführen)

### Hinweis für Windows

Ihr endgültiger DDP-Build ist eng mit der von Ihnen verwendeten mingw64-Version gekoppelt.
LLVM und die DDP-Laufzeit und stdlib müssen mit derselben Version von mingw64 erstellt werden, um zusammenzuarbeiten.
Da DDP GCC als Linker und Libc-Anbieter verwendet, benötigt der endgültige DDP-Build dieselbe mingw64-Version mit der es gebaut wurde, um DDP-Programme zu kompilieren. Denken Sie daran, wenn Sie DDP auf einem beliebigen Computer einrichten.

## Unter Linux

Nachdem Sie LLVM installiert haben, führen Sie einfach `make` im Stammverzeichnis des Repositoriums aus.
Um die Tests auszuführen, verwenden Sie `make test`.
Um LLVM aus dem Untermodul llvm-project zu erstellen, führen Sie `make llvm` aus.
Wenn Sie llvm aus dem Submodul erstellt haben, verwendet make diesen llvm-build anstelle des globalen.

## Unter Windows

Nachdem Sie LLVM installiert haben (siehe *LLVM bauen*), befolgen Sie die gleichen Schritte wie unter Linux, aber verwenden Sie git-bash, um die Befehle auszuführen (sonst funktionieren einige Unix-Befehle nicht).

### Wichtig
GNUMake wird nicht funktionieren. Wenn Sie gnu make installiert haben, hat es unter Windows den Namen *make*, daher sollten Sie stattdessen mingw32-make verwenden, wenn Sie die Makefiles ausführen.

# LLVM bauen

## Voraussetzungen

- [CMake](https://cmake.org/download/) (Version 3.13.4 oder höher)
- [Python3](https://www.python.org/downloads/)

Unter Windows sind die oben genannten Voraussetzungen ebenfalls erforderlich, und alle folgenden Befehle müssen in git-bash und nicht in cmd oder Powershell ausgeführt werden.

Nachdem Sie alle Voraussetzungen installiert haben, öffnen Sie ein Terminal im Root des Repositoriums und führen Sie `make llvm` aus.
Dadurch sollte das Untermodul llvm-project heruntergeladen und in llvm_build gebaut werden.
Der Download ist ungefähr 1 GB groß, daher kann es eine Weile dauern.
Das Bauen von LLVM dauert ebenfalls 2-3 Stunden, kann aber im Hintergrund ausgeführt werden, da es nicht viele Ressourcen beansprucht.

# Vollständiges Beispiel unter Windows
Alle befehle wurden im Root des Repositoriums ausgeführt.

```
$ make llvm
$ make
```
