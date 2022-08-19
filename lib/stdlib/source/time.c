/*
	defines inbuilt ddp functions to work with time
*/
#include <time.h>
#include "ddptypes.h"

ddpint Zeit_Seit_Programmstart() {
	return clock();
}