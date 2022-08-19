#include "main.h"
#include "ddptypes.h"

void Programm_Beenden(ddpint code) {
	end_runtime();
	exit(code);
}

void Laufzeitfehler(ddpstring* Nachricht, ddpint code) {
	runtime_error(code, Nachricht->str);
}