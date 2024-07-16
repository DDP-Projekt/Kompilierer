#include "../../../../lib/runtime/include/ddptypes.h"

typedef ddpstring *Zeichenkette;

void unary6(ddpstring *ret, Zeichenkette p1) {
	p1->str[0] = 'Q';
	*ret = *p1;
	*p1 = DDP_EMPTY_STRING;
}

ddpchar binary5(Zeichenkette a, ddpint b) {
	return (ddpchar)a->str[b];
}
