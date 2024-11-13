#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/utf8/utf8.h"
#include <string.h>

typedef struct {
	ddpstring puffer;
	ddpint laenge;
} TextBauer;

typedef TextBauer *TextBauerRef;

void Erhoehe_Kapazitaet(TextBauerRef bauer, ddpint cap) {
	bauer->puffer.str = ddp_reallocate(bauer->puffer.str, bauer->puffer.cap, cap);
	memset(&bauer->puffer.str[bauer->puffer.cap], 0, cap - bauer->puffer.cap);
	bauer->puffer.cap = cap;
}

ddpint Bauer_Ende_Zeiger(TextBauerRef bauer) {
	return (ddpint)bauer->puffer.str + bauer->laenge;
}

void TextBauer_Als_Text(ddpstring *ret, TextBauerRef bauer) {
	*ret = DDP_EMPTY_STRING;

	ret->str = DDP_ALLOCATE(char, bauer->laenge + 1);
	ret->str[bauer->laenge] = '\0';
	ret->cap = bauer->laenge + 1;

	memcpy(ret->str, bauer->puffer.str, bauer->laenge);
}

void TextBauer_Buchstabe_Anfuegen_C(TextBauerRef bauer, ddpchar c) {
	utf8_char_to_string(&bauer->puffer.str[bauer->laenge], c);
}
