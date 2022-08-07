#include "main.h"
#include "ddptypes.h"

void ddpextern_Programm_Beenden() {
	end_runtime();
	exit(0);
}

void ddpextern_Laufzeitfehler(ddpstring* Nachricht, ddpint code) {
	runtime_error(code, Nachricht->str);
}