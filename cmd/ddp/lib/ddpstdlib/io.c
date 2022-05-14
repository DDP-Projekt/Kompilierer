#include <stdio.h>
#include "ddpstring.h"

void inbuilt_Schreibe_Zahl(ddpint p1) {
	printf("%d", p1);
}

void inbuilt_Schreibe_Kommazahl(ddpfloat p1) {
	printf("%g", p1);
}

void inbuilt_Schreibe_Boolean(ddpbool p1) {
	printf(p1 ? "wahr" : "falsch");
}

void inbuilt_Schreibe_Buchstabe(ddpchar p1) {
	printf("%lc", p1);
}

void inbuilt_Schreibe_Text(ddpstring* p1) {
	printf("%.*ls", p1->len, p1->str);
}