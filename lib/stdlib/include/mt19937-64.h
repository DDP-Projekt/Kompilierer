#ifndef DDP_MT19937_64_H
#define DDP_MT19937_64_H

#include "common.h"

// generates a random number on [0, 2^64-1]-interval
uint64_t genrand64_int64(void);

/* generates a random number on [0,1]-real-interval */
double genrand64_real1(void);
/* generates a random number on [0,1)-real-interval */
double genrand64_real2(void);
/* generates a random number on (0,1)-real-interval */
double genrand64_real3(void);

#endif // DDP_MT19937_64_H
