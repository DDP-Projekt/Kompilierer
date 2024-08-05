#include "ddpmemory.h"
#include "ddptypes.h"
#include "utf8/utf8.h"

void Buchstaben_TextRef_BuchstabenListe(ddpcharlist *ret, ddpstringref text) {
	if (ddp_string_empty(text)) {
		*ret = DDP_EMPTY_LIST(ddpcharlist);
		return;
	}

	ret->len = 0;
	ret->cap = text->cap - 1;
	ret->arr = DDP_ALLOCATE(ddpchar, text->cap - 1);

	char *end = &text->str[text->cap - 1];
	for (char *it = text->str; it != end;) {
		it += utf8_string_to_char(it, (uint32_t *)&ret->arr[ret->len++]);
	}
}
