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
	return (CString)DDP_STRING_DATA(t);
}

pointer Text_Zu_Zeiger(ddpstringref t) {
	return (CString)t;
}

void Erstelle_Byte_Puffer(ddpstring *ret, ddpint n) {
	ret->len = n;
	if (n < DDP_SMALL_STRING_BUFF_SIZE) {
		return;
	}

	ret->large.str = DDP_ALLOCATE(char, n + 1);
	ret->large.cap = n + 1;
	ret->large.str[n] = '\0';
}

ddpint Text_Byte_Groesse(ddpstringref t) {
	return t->len;
}

ddpint Buchstabe_Byte_Groesse(ddpchar b) {
	return (ddpint)utf8_num_bytes_char(b);
}
