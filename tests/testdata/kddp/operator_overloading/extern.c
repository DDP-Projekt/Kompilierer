#include "DDP/ddptypes.h"

typedef ddpstring *Zeichenkette;

void unary6(ddpstring *ret, Zeichenkette p1) {
	char *str_ptr = DDP_GET_STRING_PTR(p1);
	str_ptr[0] = 'Q';
	*ret = *p1;
	*p1 = DDP_EMPTY_STRING;
}

ddpchar binary5(Zeichenkette a, ddpint b) {
	return (ddpchar)DDP_GET_STRING_PTR(a)[b];
}
