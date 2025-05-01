#include "DDP/ddptypes.h"
#include "DDP/sha-256-512.h"
#include <stdio.h>

void SHA_256(ddpstring *ret, ddpstring *text) {
	if (ddp_string_empty(text)) {
		ddp_string_from_constant(ret, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");
		return;
	}

	SHA256_HASH digest;
	Sha256Calculate(DDP_STRING_DATA(text), text->len, &digest);

	char buff[SHA256_HASH_SIZE * 2 + 1];
	// convert hash in buf to hex string and store in ret
	for (int i = 0; i < SHA256_HASH_SIZE; i++) {
		sprintf(buff + (i * 2), "%02x", digest.bytes[i]);
	}
	buff[sizeof(buff) - 1] = '\0';
	ddp_string_from_constant(ret, buff);
}

void SHA_512(ddpstring *ret, ddpstring *text) {
	if (ddp_string_empty(text)) {
		ddp_string_from_constant(ret, "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e");
		return;
	}

	SHA512_HASH digest;
	Sha512Calculate(DDP_STRING_DATA(text), text->len, &digest);

	// convert hash in buf to hex string and store in ret
	char buff[SHA512_HASH_SIZE * 2 + 1];
	// convert hash in buf to hex string and store in ret
	for (int i = 0; i < SHA512_HASH_SIZE; i++) {
		sprintf(buff + (i * 2), "%02x", digest.bytes[i]);
	}
	buff[sizeof(buff) - 1] = '\0';
	ddp_string_from_constant(ret, buff);
}
