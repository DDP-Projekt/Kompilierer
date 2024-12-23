#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/debug.h"
#include "DDP/utf8/utf8.h"
#include <string.h>

typedef struct {
	ddpstring puffer;
	ddpint laenge;
} TextBauer;

typedef TextBauer *TextBauerRef;

void Erhoehe_Kapazitaet(TextBauerRef bauer, ddpint cap) {
	DDP_DBGLOG("Erhoehe_Kapazitaet");
	if (cap <= DDP_SMALL_STRING_LIMIT) {
		return;
	}

	if (DDP_IS_SMALL_STRING(&bauer->puffer)) {
		char *newBuffer = DDP_ALLOCATE(char, cap);
		memcpy(newBuffer, DDP_GET_STRING_PTR(&bauer->puffer), bauer->puffer.cap);
		bauer->puffer.refc = NULL;
		bauer->puffer.str = newBuffer;
	} else {
		bauer->puffer.str = ddp_reallocate(bauer->puffer.str, bauer->puffer.cap, cap);
		memset(&bauer->puffer.str[bauer->puffer.cap], 0, cap - bauer->puffer.cap);
		bauer->puffer.cap = cap;
	}
}

ddpint Bauer_Ende_Zeiger(TextBauerRef bauer) {
	return (ddpint)DDP_GET_STRING_PTR(&bauer->puffer) + bauer->laenge;
}

void TextBauer_Als_Text(ddpstring *ret, TextBauerRef bauer) {
	DDP_DBGLOG("TextBauer_Als_Text");
	*ret = DDP_EMPTY_STRING;

	ret->cap = bauer->laenge + 1;
	if (DDP_IS_SMALL_STRING(ret)) {
		DDP_DBGLOG("copying small byte buffer (%lld)", bauer->laenge);
		char *ret_ptr = DDP_GET_STRING_PTR(ret);
		memcpy(ret_ptr, DDP_GET_STRING_PTR(&bauer->puffer), bauer->laenge);
		ret_ptr[bauer->laenge] = '\0';
	} else {
		DDP_DBGLOG("copying large byte buffer");
		ret->str = DDP_ALLOCATE(char, bauer->laenge + 1);
		ret->str[bauer->laenge] = '\0';
		ret->cap = bauer->laenge + 1;

		memcpy(ret->str, bauer->puffer.str, bauer->laenge);
	}
}

void TextBauer_Buchstabe_Anfuegen_C(TextBauerRef bauer, ddpchar c) {
	utf8_char_to_string(&bauer->puffer.str[bauer->laenge], c);
}
