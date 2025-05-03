#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/debug.h"
#include "DDP/error.h"
#include "DDP/utf8/utf8.h"

static void ddp_string_append(ddpstringref text, ddpstring *elm) {
	const ddpint newLen = text->len + elm->len;
	const ddpint newCap = newLen + 1;
	if (newCap <= DDP_STRING_CAP(text)) {
		DDP_DBGLOG("enough capacity");
		ddp_strncat(text, DDP_STRING_DATA(elm), elm->len);
		return;
	}

	DDP_DBGLOG("reserving capacity");
	ddp_reserve_string_capacity(text, DDP_GROW_CAPACITY(newCap));
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
