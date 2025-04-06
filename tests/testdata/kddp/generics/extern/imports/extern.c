#include "DDP/ddptypes.h"

typedef struct {
	void *arr;	// the element array
	ddpint len; // the length of the array
	ddpint cap; // the capacity of the array
} generic_list;

ddpint bar(generic_list *l) {
	return l->len;
}

ddpbool baz(void *a, void *b) {
	return a == b;
}
