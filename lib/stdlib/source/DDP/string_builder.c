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
	bauer->puffer.large.str = ddp_reallocate(bauer->puffer.large.str, bauer->puffer.large.cap, cap);
	memset(&bauer->puffer.large.str[bauer->puffer.large.cap], 0, cap - bauer->puffer.large.cap);
	bauer->puffer.large.cap = cap;
}

ddpint Bauer_Ende_Zeiger(TextBauerRef bauer) {
	return (ddpint)bauer->puffer.large.str + bauer->laenge;
}

void TextBauer_Als_Text(ddpstring *ret, TextBauerRef bauer) {
	*ret = DDP_EMPTY_STRING;

	ddp_strncat(ret, bauer->puffer.large.str, bauer->laenge);
}

void TextBauer_Buchstabe_Anfuegen_C(TextBauerRef bauer, ddpchar c) {
	utf8_char_to_string(&bauer->puffer.large.str[bauer->laenge], c);
}
