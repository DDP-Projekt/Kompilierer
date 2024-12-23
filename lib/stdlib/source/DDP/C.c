#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/utf8/utf8.h"
#include <memory.h>

typedef ddpint pointer;

void C_Memcpy(pointer dest, pointer src, ddpint size) {
	memcpy((void *)dest, (void *)src, (size_t)size);
}

typedef pointer CString;

CString Text_Zu_CString(ddpstringref t) {
	return (CString)DDP_GET_STRING_PTR(t);
}

pointer Text_Zu_Zeiger(ddpstringref t) {
	return (CString)t;
}

void Erstelle_Byte_Puffer(ddpstring *ret, ddpint n) {
	ret->cap = n + 1;
	if (!DDP_IS_SMALL_STRING(ret)) {
		ret->refc = NULL;
		ret->str = DDP_ALLOCATE(char, n + 1);
	}
	DDP_GET_STRING_PTR(ret)
	[n] = '\0';
}

ddpint Text_Byte_Groesse(ddpstringref t) {
	return t->cap > 0 ? t->cap - 1 : 0;
}

ddpint Buchstabe_Byte_Groesse(ddpchar b) {
	return (ddpint)utf8_num_bytes_char(b);
}
