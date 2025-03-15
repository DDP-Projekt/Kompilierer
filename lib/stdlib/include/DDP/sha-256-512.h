////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  WjCryptLib_Sha256 and WjCryptLib_Sha512
//
//  Implementation of SHA256 and SHA512 hash function.
//  Original author: Tom St Denis, tomstdenis@gmail.com, http://libtom.org
//  Modified by WaterJuice retaining Public Domain license.
//
//  This is free and unencumbered software released into the public domain - June 2013 waterjuice.org
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

#pragma once

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  IMPORTS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

#include <stdint.h>
#include <stdio.h>

typedef struct
{
	uint64_t length;
	uint32_t state[8];
	uint32_t curlen;
	uint8_t buf[64];
} Sha256Context;

#define SHA256_HASH_SIZE (256 / 8)

typedef struct
{
	uint8_t bytes[SHA256_HASH_SIZE];
} SHA256_HASH;

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  PUBLIC FUNCTIONS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//	Sha256Initialise
//
//	Initialises a SHA256 Context. Use this to initialise/reset a context.
void Sha256Initialise(Sha256Context *Context);

//  Sha256Update
//
//  Adds data to the SHA256 context. This will process the data and update the internal state of the context. Keep on
//  calling this function until all the data has been added. Then call Sha256Finalise to calculate the hash.
void Sha256Update(Sha256Context *Context, void const *Buffer, uint32_t BufferSize);

//  Sha256Finalise
//
//  Performs the final calculation of the hash and returns the digest (32 byte buffer containing 256bit hash). After
//  calling this, Sha256Initialised must be used to reuse the context.
void Sha256Finalise(Sha256Context *Context, SHA256_HASH *Digest);

//  Sha256Calculate
//
//  Combines Sha256Initialise, Sha256Update, and Sha256Finalise into one function. Calculates the SHA256 hash of the
//  buffer.
void Sha256Calculate(void const *Buffer, uint32_t BufferSize, SHA256_HASH *Digest);

typedef struct
{
	uint64_t length;
	uint64_t state[8];
	uint32_t curlen;
	uint8_t buf[128];
} Sha512Context;

#define SHA512_HASH_SIZE (512 / 8)

typedef struct {
	uint8_t bytes[SHA512_HASH_SIZE];
} SHA512_HASH;

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  PUBLIC FUNCTIONS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//  Sha512Initialise
//
//  Initialises a SHA512 Context. Use this to initialise/reset a context.
void Sha512Initialise(Sha512Context *Context);

//  Sha512Update
//
//  Adds data to the SHA512 context. This will process the data and update the internal state of the context. Keep on
//  calling this function until all the data has been added. Then call Sha512Finalise to calculate the hash.
void Sha512Update(Sha512Context *Context, void const *Buffer, uint32_t BufferSize);

//  Sha512Finalise
//
//  Performs the final calculation of the hash and returns the digest (64 byte buffer containing 512bit hash). After
//  calling this, Sha512Initialised must be used to reuse the context.
void Sha512Finalise(Sha512Context *Context, SHA512_HASH *Digest);

//  Sha512Calculate
//
//  Combines Sha512Initialise, Sha512Update, and Sha512Finalise into one function. Calculates the SHA512 hash of the
//  buffer.
void Sha512Calculate(void const *Buffer, uint32_t BufferSize, SHA512_HASH *Digest);