#include <time.h>
#include "ddptypes.h"

ddpint inbuilt_Zeit_Seit_Programmstart() {
	return clock();
}