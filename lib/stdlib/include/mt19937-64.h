#ifndef DDP_MT19937_64_H
#define DDP_MT19937_64_H

#include "common.h"

// generates a random number on [0, 2^64-1]-interval
uint64_t genrand64_int64();

/* generates a random number on [0,1]-real-interval */
double genrand64_real1();
/* generates a random number on [0,1)-real-interval */
double genrand64_real2();
/* generates a random number on (0,1)-real-interval */
double genrand64_real3();

#endif // DDP_MT19937_64_H