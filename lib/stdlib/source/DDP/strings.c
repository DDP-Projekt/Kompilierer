#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/debug.h"
#include "DDP/error.h"
#include "DDP/utf8/utf8.h"
#include <string.h>

static void ddp_string_append(ddpstringref text, ddpstring *elm) {
	const ddpint newLen = text->len + elm->len;
	const ddpint newCap = newLen + 1;
	if (newCap <= DDP_STRING_CAP(text)) {
		ddp_strncat(text, DDP_STRING_DATA(elm), elm->len);
		return;
	}

	if (DDP_IS_SMALL_STRING(text)) {
		// allocate some more to prevent future allocations
		const size_t cap = DDP_MAX(newCap, DDP_SMALL_ALLOCATION_SIZE);
		char *string = DDP_ALLOCATE(char, cap); // the char array of the string (plus null terminator)
		memcpy(string, DDP_STRING_DATA(text), text->len);
		memcpy(&string[text->len], DDP_STRING_DATA(elm), elm->len + 1);

		text->len = newLen;
		text->large.cap = cap;
		text->large.str = string;
		return;
	}

	size_t cap = DDP_GROW_CAPACITY(DDP_STRING_CAP(text));
	while (cap < newCap) {
		cap = DDP_GROW_CAPACITY(cap);
	}

	text->large.str = ddp_reallocate(text->large.str, text->large.cap, cap);
	text->large.cap = cap;
	ddp_strncat(text, DDP_STRING_DATA(elm), elm->len);
}

void Text_An_Text_Fügen(ddpstringref text, ddpstring *elm) {
	DDP_DBGLOG("Text_An_Text");
	ddp_string_append(text, elm);
}

void Buchstabe_An_Text_Fügen(ddpstringref text, ddpchar elm) {
	ddpstring s;
	s.len = utf8_char_to_string(s.small.str, elm);
	if (s.len < 1) {
		ddp_error("Invalider Buchstabe: %d", false, (int)elm);
		return;
	}
	ddp_string_append(text, &s);
}
