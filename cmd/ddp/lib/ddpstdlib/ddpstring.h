#ifndef DDPSTRING_H
#define DDPSTRING_H

#include "gc.h"

ddpstring* inbuilt_string_from_constant(ddpchar* str, ddpint len);
void inbuilt_free_string(ddpstring* str);
ddpstring* inbuilt_deep_copy_string(ddpstring* str);

#endif // DDPSTRING_H