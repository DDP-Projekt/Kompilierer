#include "sha-256-512.h"
#include "ddptypes.h"
#include "stdio.h"
#include "ddpmemory.h"

void SHA_256(ddpstring *ret, ddpstring *text) {
	SHA256_HASH digest;
	Sha256Calculate((char*)text->str, text->cap-1, &digest);

	// convert hash in buf to hex string and store in ret
	ret->cap = SHA256_HASH_SIZE * 2;
	ret->str = ALLOCATE(char, ret->cap);
	for (int i = 0; i < SHA256_HASH_SIZE; i++) {
		sprintf(ret->str + (i * 2), "%02x", digest.bytes[i]);
	}
}

void SHA_512(ddpstring *ret, ddpstring *text) {
	SHA512_HASH digest;
	Sha512Calculate((char*)text->str, text->cap-1, &digest);

	// convert hash in buf to hex string and store in ret
	ret->cap = SHA512_HASH_SIZE * 2;
	ret->str = ALLOCATE(char, ret->cap);
	for (int i = 0; i < SHA512_HASH_SIZE; i++) {
		sprintf(ret->str + (i * 2), "%02x", digest.bytes[i]);
	}
}