#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include <string.h>

void Text_Zu_ByteListe(ddpbytelist *ret, ddpstringref t) {
	if (ddp_string_empty(t)) {
		*ret = DDP_EMPTY_LIST(ddpbytelist);
		return;
	}
	ret->cap = t->cap - 1;
	ret->len = ret->cap;
	ret->arr = DDP_ALLOCATE(ddpbyte, ret->len);
	memcpy(ret->arr, t->str, ret->len);
}

void ByteListe_Zu_Text(ddpstring *ret, ddpbytelistref b) {
	if (b->len == 0) {
		*ret = DDP_EMPTY_STRING;
		return;
	}

	ret->cap = b->len + 1;
	ret->str = DDP_ALLOCATE(char, ret->cap);
	memcpy(ret->str, b->arr, b->len);
	ret->str[b->len] = '\0';
}
