#include "main.h"
#include "ddptypes.h"

void ddpextern_Programm_Beenden(ddpint code) {
	end_runtime();
	exit(code);
}

void ddpextern_Laufzeitfehler(ddpstring* Nachricht, ddpint code) {
	runtime_error(code, Nachricht->str);
}