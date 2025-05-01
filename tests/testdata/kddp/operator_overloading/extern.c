#include "DDP/ddptypes.h"

typedef ddpstring *Zeichenkette;

void unary6(ddpstring *ret, Zeichenkette p1) {
	DDP_STRING_DATA(p1)
	[0] = 'Q';
	*ret = *p1;
	*p1 = DDP_EMPTY_STRING;
}

ddpchar binary5(Zeichenkette a, ddpint b) {
	return (ddpchar)DDP_STRING_DATA(a)[b];
}
