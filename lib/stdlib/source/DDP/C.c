#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include "DDP/utf8/utf8.h"
#include <memory.h>

typedef ddpint pointer;

void C_Memcpy(pointer dest, pointer src, ddpint size) {
	memcpy((void *)dest, (void *)src, (size_t)size);
}

typedef pointer C_String;

C_String Text_Zu_C_String(ddpstringref t) {
	return (C_String)t->str;
}

pointer Text_Zu_Zeiger(ddpstringref t) {
	return (C_String)t;
}

void Erstelle_Byte_Puffer(ddpstring *ret, ddpint n) {
	ret->str = DDP_ALLOCATE(char, n + 1);
	ret->cap = n + 1;
	ret->str[n] = '\0';
}

ddpint Text_Byte_Groesse(ddpstringref t) {
	return t->cap - 1;
}

ddpint Buchstabe_Byte_Groesse(ddpchar b) {
	return (ddpint)utf8_num_bytes_char(b);
}
