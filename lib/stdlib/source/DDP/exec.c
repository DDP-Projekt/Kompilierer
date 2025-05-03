#include "DDP/ddpmemory.h"
#include "DDP/ddpos.h"
#include "DDP/ddptypes.h"
#include "DDP/error.h"
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#if DDPOS_WINDOWS
#include "DDP/ddpwindows.h"
#else // DDPOS_LINUX
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

#endif // DDPOS_WINDOWS

#define READ_END 0
#define WRITE_END 1
#define BUFF_SIZE 512

#if DDPOS_WINDOWS

// creates a pipe and sets the inherit_handle to be inherited
static bool create_pipe(HANDLE pipe_handles[], int inherit_handle) {
	return CreatePipe(&pipe_handles[READ_END], &pipe_handles[WRITE_END], NULL, 0) &&
		   SetHandleInformation(pipe_handles[inherit_handle], HANDLE_FLAG_INHERIT, HANDLE_FLAG_INHERIT);
}

static void close_pipe(HANDLE pipe_handles[]) {
	CloseHandle(pipe_handles[0]);
	CloseHandle(pipe_handles[1]);
}

// reads everything from the given pipe into out
// then closes the pipe
// returns the new size of out
static void read_pipe(HANDLE handle, ddpstringref out) {
	ddp_free_string(out);

	char buff[BUFF_SIZE];
	DWORD nread = 0;
	while (ReadFile(handle, buff, BUFF_SIZE, &nread, 0)) {
		ddp_strncat(out, buff, nread);
	}
	CloseHandle(handle);
}

// TODO: use err
static ddpint execute_process(ddpstring *path, ddpstringlist *args,
							  ddpstring *input, ddpstringref stdoutput, ddpstringref erroutput) {
	HANDLE stdout_pipe[2];
	HANDLE stderr_pipe[2];
	HANDLE stdin_pipe[2];

	const bool need_stderr = stdoutput != erroutput;

	// for stdout and stderr we want to inherit the write end to the child, as it writes to those pipes
	// for stdin we inherit the read end because the child because it reads from it
	if (!create_pipe(stdout_pipe, WRITE_END)) {
		ddp_error_win("Fehler beim Öffnen der Pipe: ");
		return -1;
	}
	if (need_stderr && !create_pipe(stderr_pipe, WRITE_END)) {
		ddp_error_win("Fehler beim Öffnen der Pipe: ");
		close_pipe(stdout_pipe);
		return -1;
	}
	if (!create_pipe(stdin_pipe, READ_END)) {
		ddp_error_win("Fehler beim Öffnen der Pipe: ");
		close_pipe(stdout_pipe);
		if (need_stderr) {
			close_pipe(stderr_pipe);
		}
		return -1;
	}

	// prepare the arguments
	char *argv;
	size_t argv_size = path->len + 1;
	for (ddpint i = 0; i < args->len; i++) {
		argv_size += args->arr[i].len + 1; // the nullterminator is used for the trailing space
	}
	argv = DDP_ALLOCATE(char, argv_size);
	argv[0] = '\0'; // make sure strcat works
	strcat(argv, DDP_STRING_DATA(path));
	strcat(argv, " ");
	for (ddpint i = 0; i < args->len; i++) {
		strcat(argv, DDP_STRING_DATA(&args->arr[i]));
		if (i < args->len - 1) {
			strcat(argv, " ");
		}
	}

	// setup what to inherit for the child process
	STARTUPINFOA si = {0};
	si.cb = sizeof(si);
	si.hStdInput = stdin_pipe[READ_END];
	si.hStdOutput = stdout_pipe[WRITE_END];
	si.hStdError = need_stderr ? stderr_pipe[WRITE_END] : stdout_pipe[WRITE_END];
	si.dwFlags |= STARTF_USESTDHANDLES;

	// start the actual child process
	PROCESS_INFORMATION pi;
	if (!CreateProcessA(DDP_STRING_DATA(path), argv, NULL, NULL, true, 0, NULL, NULL, &si, &pi)) {
		ddp_error_win("Fehler beim Erstellen des Unter Prozesses: ");
		close_pipe(stdout_pipe);
		if (need_stderr) {
			close_pipe(stderr_pipe);
		}
		close_pipe(stdin_pipe);
		DDP_FREE_ARRAY(char, argv, argv_size); // free the arguments
		return -1;
	}
	DDP_FREE_ARRAY(char, argv, argv_size); // free the arguments

	// you NEED to close these, or it will not work
	CloseHandle(pi.hThread);
	CloseHandle(stdout_pipe[WRITE_END]);
	if (need_stderr) {
		CloseHandle(stderr_pipe[WRITE_END]);
	}
	CloseHandle(stdin_pipe[READ_END]);

	// write stdin
	DWORD len_written = 0;
	DWORD len_to_write = input->len;
	if (!WriteFile(stdin_pipe[WRITE_END], DDP_STRING_DATA(input), len_to_write, &len_written, NULL) || len_written != len_to_write) {
		ddp_error_win("Fehler beim schreiben der Eingabe: ");
		// terminate the running process
		TerminateProcess(pi.hProcess, 1);
		CloseHandle(pi.hProcess);

		CloseHandle(stdout_pipe[READ_END]);
		if (need_stderr) {
			CloseHandle(stderr_pipe[READ_END]);
		}
		CloseHandle(stdin_pipe[WRITE_END]);
		return -1;
	}
	CloseHandle(stdin_pipe[WRITE_END]);

	// read stdout and stderr if needed
	read_pipe(stdout_pipe[READ_END], stdoutput);

	if (need_stderr) {
		read_pipe(stderr_pipe[READ_END], erroutput);
	}

	WaitForSingleObject(pi.hProcess, INFINITE);
	DWORD exit_code;
	GetExitCodeProcess(pi.hProcess, &exit_code);
	CloseHandle(pi.hProcess);
	return (ddpint)exit_code;
}
#else // DDPOS_LINUX

#define COMMAND_NOT_FOUND 127

// reads everything from the given pipe into out
// then closes the pipe
// returns the new size of out
static void read_pipe(int fd, ddpstringref out) {
	ddp_free_string(out);

	*out = DDP_EMPTY_STRING;
	char buff[BUFF_SIZE];
	int nread;
	while ((nread = read(fd, buff, sizeof(buff))) > 0) {
		ddp_strncat(out, buff, nread);
	}
	close(fd);
}

// executes path with the given args
// pipes the given input to the processes stdin
// returns the processes stdout and stderr into the given stdoutput
// and erroutput out-variables
// erroutput may be equal to stdoutput if they shall be read together
// but not NULL
static ddpint execute_process(ddpstring *path, ddpstringlist *args,
							  ddpstring *input, ddpstringref stdoutput, ddpstringref erroutput) {
	int stdout_fd[2];
	int stderr_fd[2];
	int stdin_fd[2];

	const bool need_stderr = stdoutput != erroutput;

	// prepare the pipes
	if (pipe(stdout_fd)) {
		ddp_error("Fehler beim Öffnen der Pipe: ", true);
		return -1;
	}
	if (need_stderr && pipe(stderr_fd)) {
		ddp_error("Fehler beim Öffnen der Pipe: ", true);
		close(stdout_fd[0]);
		close(stdout_fd[1]);
		return -1;
	}
	if (pipe(stdin_fd)) {
		ddp_error("Fehler beim Öffnen der Pipe: ", true);
		close(stdout_fd[0]);
		close(stdout_fd[1]);
		if (need_stderr) {
			close(stderr_fd[0]);
			close(stderr_fd[1]);
		}
		return -1;
	}

	// prepare the arguments
	const size_t argc = args->len + 1;
	char **process_args = DDP_ALLOCATE(char *, argc + 1); // + 1 for the terminating NULL

	process_args[0] = DDP_ALLOCATE(char, path->len + 1);
	strcpy(process_args[0], DDP_STRING_DATA(path));
	for (int i = 1; i < argc; i++) {
		process_args[i] = DDP_ALLOCATE(char, args->arr[i - 1].len + 1);
		strcpy(process_args[i], DDP_STRING_DATA(&args->arr[i - 1]));
	}
	process_args[argc] = NULL;

	// create the supprocess
	switch (fork()) {
	case -1: // error
		ddp_error("Fehler beim Erstellen des Unter Prozesses: ", true);
		return -1;
	case 0: { // child
		close(stdout_fd[READ_END]);
		if (need_stderr) {
			close(stderr_fd[READ_END]);
		}
		close(stdin_fd[WRITE_END]);
		dup2(stdout_fd[WRITE_END], STDOUT_FILENO);
		dup2(need_stderr ? stderr_fd[WRITE_END] : stdout_fd[WRITE_END], STDERR_FILENO);
		dup2(stdin_fd[READ_END], STDIN_FILENO);
		execvp(DDP_STRING_DATA(path), process_args);
		fprintf(stderr, "Fehler beim Starten des Unter Prozesses: %s", strerror(errno));
		exit(COMMAND_NOT_FOUND);
		return -1;
	}
	default: { // parent
		// free the arguments
		for (int i = 0; i < argc; i++) {
			DDP_FREE_ARRAY(char, process_args[i], strlen(process_args[i]) + 1);
		}
		DDP_FREE_ARRAY(char *, process_args, argc + 1);

		close(stdout_fd[WRITE_END]);
		if (need_stderr) {
			close(stderr_fd[WRITE_END]);
		}
		close(stdin_fd[READ_END]);

		if (write(stdin_fd[WRITE_END], DDP_STRING_DATA(input), input->len) < 0) {
			ddp_error("Fehler beim schreiben der Eingabe: ", true);
			return -1;
		}
		close(stdin_fd[WRITE_END]);

		int exit_code;
		if (wait(&exit_code) == -1) {
			ddp_error("Fehler beim Warten auf den Unter Prozess: ", true);
			return -1;
		}

		if (WIFEXITED(exit_code) && WEXITSTATUS(exit_code) == COMMAND_NOT_FOUND) {
			read_pipe((need_stderr ? stderr_fd : stdout_fd)[READ_END], erroutput);
			ddp_error("Fehler beim Starten des Unter Prozesses: %s", false, DDP_STRING_DATA(erroutput));
			ddp_free_string(erroutput);
			*erroutput = (ddpstring){0};
			return -1;
		}

		read_pipe(stdout_fd[READ_END], stdoutput);

		if (need_stderr) {
			read_pipe(stderr_fd[READ_END], erroutput);
		}

		if (WIFEXITED(exit_code)) {
			return (ddpint)WEXITSTATUS(exit_code);
		}
		return -1;
	}
	}
}

#endif // DDPOS_WINDOWS

ddpint Programm_Ausfuehren(ddpstring *ProgrammName, ddpstringlist *Argumente,
						   ddpstring *StandardEingabe, ddpstringref StandardAusgabe, ddpstringref StandardFehlerAusgabe) {
	DDP_MIGHT_ERROR;
	return execute_process(ProgrammName, Argumente, StandardEingabe, StandardAusgabe, StandardFehlerAusgabe);
}
