#include "DDP/ddptypes.h"

/*
    The extern functions below require Duden/Fehlerbehandlung
    to be imported in the DDP code
*/

extern ddpbool Gab_Fehler(void);
extern void Loesche_Fehler(void);
extern void Letzter_Fehler(ddpstringref Fehler);
// frees Fehler
extern void Setze_Fehler(ddpstring *Fehler);

// macro for readability
// supposed to be first statement in functions that might report errors
#define DDP_MIGHT_ERROR Loesche_Fehler()

// utility function for use in the c-source of the stdlib
//
// passes an OS-specific error message prefixed with prefix
// to Setzte_Fehler
void ddp_error(const char *prefix, bool use_errno, ...);

#if DDPOS_WINDOWS
// same as ddp_error, but uses GetLastError() instead of errno
void ddp_error_win(const char *prefix, ...);
#endif
