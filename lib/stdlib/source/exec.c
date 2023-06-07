#include "ddptypes.h"
#include "ddpos.h"
#include "memory.h"
#include <stdarg.h>

#if DDPOS_WINDOWS
#error Not Implemented
#else // DDPOS_LINUX
#include <unistd.h>
#include <sys/wait.h>
#include <sys/types.h>
#endif // DDPOS_WINDOWS

#define READ_END 0
#define WRITE_END 1
#define BUFF_SIZE 512

#if DDPOS_WINDOWS
// TODO: add execute_process using the WinApi
#else // DDPOS_LINUX

// helper to output an error
static void write_error(ddpstringref ref, const char* fmt, ...) {
	char errbuff[1024];

	va_list argptr;
	va_start(argptr, fmt);

	int len = vsprintf(errbuff, fmt, argptr);
	
	va_end(argptr);

	ref->str = ddp_reallocate(ref->str, ref->cap, len+1);
	memcpy(ref->str, errbuff, len);
	ref->cap = len+1;
	ref->str[ref->cap-1] = '\0';
}

// reads everything from the given pipe into out
// then closes the pipe
// returns the new size of out
static size_t read_pipe(int fd, char** out) {
        size_t out_size = 0;
        char buff[BUFF_SIZE];
        int nread;
        while ((nread = read(fd, buff, sizeof(buff))) > 0) {
            *out = ddp_reallocate(*out, out_size, out_size + nread);
            memcpy(*out + out_size, buff, nread);
            out_size += nread;
        }
        close(fd);
        return out_size;
}

// executes path with the given args
// pipes the given input to the processes stdin
// returns the processes stdout and stderr into the given stdoutput
// and erroutput out-variables
// erroutput may be equal to stdoutput if they shall be read together
// but not NULL
static ddpint execute_process(ddpstring* path, ddpstringlist* args, ddpstringref err,
    ddpstring* input, ddpstringref stdoutput, ddpstringref erroutput)
{
    int stdout_fd[2];
    int stderr_fd[2];
    int stdin_fd[2];

    const bool need_stderr = stdoutput != erroutput;

    // prepare the pipes
    if (pipe(stdout_fd)) {
        write_error(err, "Fehler beim Öffnen der Pipe: %s", strerror(errno));
        return -1;
    }
    if (need_stderr && pipe(stderr_fd)) {
        write_error(err, "Fehler beim Öffnen der Pipe: %s", strerror(errno));
        close(stdout_fd[0]);
        close(stdout_fd[1]);
        return -1;
    }
    if (pipe(stdin_fd)) {
        write_error(err, "Fehler beim Öffnen der Pipe: %s", strerror(errno));
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
    char** process_args = ALLOCATE(char*, argc+1); // + 1 for the terminating NULL

    process_args[0] = ALLOCATE(char, strlen(path->str)+1);
    strcpy(process_args[0], path->str);
    for (int i = 1; i < argc; i++) {
        process_args[i] = ALLOCATE(char, strlen(args->arr[i-1].str)+1);
        strcpy(process_args[i], args->arr[i-1].str);
    }
    process_args[argc] = NULL;

    // prepare the output
    FREE_ARRAY(char, stdoutput->str, stdoutput->cap);
    stdoutput->cap = 0;
    stdoutput->str = NULL;
    if (need_stderr) {
        FREE_ARRAY(char, erroutput->str, erroutput->cap);
        erroutput->cap = 0;
        erroutput->str = NULL;
    }

    // create the supprocess
    switch (fork()) {
    case -1: // error
        write_error(err, "Fehler beim Erstellen des Unter Prozesses: %s", strerror(errno));
        return -1;
    case 0: { // child
        close(stdout_fd[READ_END]);
        if (need_stderr)
            close(stderr_fd[READ_END]);
        close(stdin_fd[WRITE_END]);
        dup2(stdout_fd[WRITE_END], STDOUT_FILENO);
        dup2(need_stderr ? stderr_fd[WRITE_END] : stdout_fd[WRITE_END], STDERR_FILENO);
        dup2(stdin_fd[READ_END], STDIN_FILENO);
        execv(path->str, process_args);
        write_error(err, "Fehler beim Starten des Unter Prozesses: %s", strerror(errno));
        return -1;
    }
    default: { // parent
        // free the arguments
        for (int i = 0; i < argc; i++) {
            FREE_ARRAY(char, process_args[i], strlen(process_args[i])+1);
        }
        FREE_ARRAY(char*, process_args, argc+1);
        
        close(stdout_fd[WRITE_END]);
        if (need_stderr)
            close(stderr_fd[WRITE_END]);
        close(stdin_fd[READ_END]);

        if (write(stdin_fd[WRITE_END], input->str, input->cap) < 0) {
            return -1;
        }
        close(stdin_fd[WRITE_END]);

        int exit_code;
        if (wait(&exit_code) == -1) {
            write_error(err, "Fehler beim Warten auf den Unter Prozess: %s", strerror(errno));
            return -1;
        }

        size_t stdout_size = read_pipe(stdout_fd[READ_END], &stdoutput->str);
        stdoutput->str = ddp_reallocate(stdoutput->str, stdout_size, stdout_size+1);
        stdoutput->str[stdout_size] = '\0';
        stdoutput->cap = stdout_size+1;

        if (need_stderr) {
            size_t stderr_size = read_pipe(stderr_fd[READ_END], &erroutput->str);
            erroutput->str = ddp_reallocate(erroutput->str, stderr_size, stderr_size+1);
            erroutput->str[stderr_size] = '\0';
            erroutput->cap = stderr_size+1;
        }

        if (WIFEXITED(exit_code)) {
            return (ddpint)WEXITSTATUS(exit_code);
        }
        return -1;
    }
    }
}

#endif // DDPOS_WINDOWS

ddpint Programm_Ausfuehren(ddpstring* ProgrammName, ddpstringlist* Argumente, ddpstringref Fehler,
    ddpstring* StandardEingabe, ddpstringref StandardAusgabe, ddpstringref StandardFehlerAusgabe) {
    // clear error
    ddp_reallocate(Fehler->str, Fehler->cap, 1);
    Fehler->str[0] = '\0';
    return execute_process(ProgrammName, Argumente, Fehler, StandardEingabe, StandardAusgabe, StandardFehlerAusgabe);
}