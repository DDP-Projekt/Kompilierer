/*
	defines inbuilt ddp functions to work with io
*/
#include "ddptypes.h"
#include "utf8/utf8.h"

void inbuilt_Schreibe_Zahl(ddpint p1) {
	printf("%ld", p1);
}

void inbuilt_Schreibe_Kommazahl(ddpfloat p1) {
	printf("%.16g", p1);
}

void inbuilt_Schreibe_Boolean(ddpbool p1) {
	printf(p1 ? "wahr" : "falsch");
}

void inbuilt_Schreibe_Buchstabe(ddpchar p1) {
	char temp[5];
	utf8_char_to_string(temp, p1);
	printf("%s", temp);
}

void inbuilt_Schreibe_Text(ddpstring* p1) {
	printf("%s", p1->str);
}