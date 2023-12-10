#include "ddptypes.h"

extern ddpbool Gab_Fehler();
extern void Loesche_Fehler();
extern void Letzter_Fehler(ddpstringref Fehler);
// frees Fehler
extern void Setze_Fehler(ddpstring* Fehler);

// utility function for use in the c-source of the stdlib
//
// passes an OS-specific error message prefixed with prefix
// to Setzte_Fehler
void ddp_error(const char* prefix, bool use_errno);

#if DDPOS_WINDOWS
void ddp_error_win(const char* prefix);
#endif