#include <time.h>
#include "gc.h"

ddpint inbuilt_Zeit_Seit_Programmstart() {
	return clock();
}