#include "ddptypes.h"
#include <math.h>

ddpfloat Sinus(ddpfloat x) {
	return sin((double)x);
}

ddpfloat Kosinus(ddpfloat x) {
	return cos((double)x);
}

ddpfloat Tangens(ddpfloat x) {
	return tan((double)x);
}

ddpfloat Arkussinus(ddpfloat x) {
	return asin((double)x);
}

ddpfloat Arkuskosinus(ddpfloat x) {
	return acos((double)x);
}

ddpfloat Arkustangens(ddpfloat x) {
	return atan((double)x);
}

ddpfloat Hyperbelsinus(ddpfloat x) {
	return sinh((double)x);
}

ddpfloat Hyperbelkosinus(ddpfloat x) {
	return cosh((double)x);
}

ddpfloat Hyperbeltangens(ddpfloat x) {
	return tanh((double)x);
}

ddpfloat Areahyperbelsinus(ddpfloat x) {
	return asinh((double)x);
}

ddpfloat Areahyperbelkosinus(ddpfloat x) {
	return acosh((double)x);
}

ddpfloat Areahyperbeltangens(ddpfloat x) {
	return atanh((double)x);
}

ddpfloat Winkel(ddpfloat x, ddpfloat y) {
	return atan2((double)x, (double)y);
}

ddpfloat Gausssche_Fehlerfunktion(ddpfloat x) {
	return erf((double)x);
}
