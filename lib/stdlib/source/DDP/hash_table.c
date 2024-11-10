#include "DDP/ddptypes.h"
#include <stdint.h>

#define MAGIC_PRIME 0x01000193
#define INITIAL_HASH 0x811c9dc5

ddpint FNV_Hash(ddpstringref str) {
	uint32_t hash = INITIAL_HASH;

	uint8_t *bytes = (uint8_t *)str->str;
	while (*bytes++) {
		hash = (hash ^ *bytes) * MAGIC_PRIME;
	}

	return (ddpint)hash;
}
