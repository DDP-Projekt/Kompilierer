////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  IMPORTS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

#include "sha-256-512.h"
#include <memory.h>

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  MACROS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

#define ror(value, bits) (((value) >> (bits)) | ((value) << (32 - (bits))))

#define MIN(x, y) ( ((x)<(y))?(x):(y) )

#define STORE32H(x, y)                                                                     \
     { (y)[0] = (uint8_t)(((x)>>24)&255); (y)[1] = (uint8_t)(((x)>>16)&255);   \
       (y)[2] = (uint8_t)(((x)>>8)&255); (y)[3] = (uint8_t)((x)&255); }

#define LOAD32H(x, y)                            \
     { x = ((uint32_t)((y)[0] & 255)<<24) | \
           ((uint32_t)((y)[1] & 255)<<16) | \
           ((uint32_t)((y)[2] & 255)<<8)  | \
           ((uint32_t)((y)[3] & 255)); }

#define STORE64H(x, y)                                                                     \
   { (y)[0] = (uint8_t)(((x)>>56)&255); (y)[1] = (uint8_t)(((x)>>48)&255);     \
     (y)[2] = (uint8_t)(((x)>>40)&255); (y)[3] = (uint8_t)(((x)>>32)&255);     \
     (y)[4] = (uint8_t)(((x)>>24)&255); (y)[5] = (uint8_t)(((x)>>16)&255);     \
     (y)[6] = (uint8_t)(((x)>>8)&255); (y)[7] = (uint8_t)((x)&255); }

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  CONSTANTS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// The K array
static const uint32_t K256[64] = {
    0x428a2f98UL, 0x71374491UL, 0xb5c0fbcfUL, 0xe9b5dba5UL, 0x3956c25bUL,
    0x59f111f1UL, 0x923f82a4UL, 0xab1c5ed5UL, 0xd807aa98UL, 0x12835b01UL,
    0x243185beUL, 0x550c7dc3UL, 0x72be5d74UL, 0x80deb1feUL, 0x9bdc06a7UL,
    0xc19bf174UL, 0xe49b69c1UL, 0xefbe4786UL, 0x0fc19dc6UL, 0x240ca1ccUL,
    0x2de92c6fUL, 0x4a7484aaUL, 0x5cb0a9dcUL, 0x76f988daUL, 0x983e5152UL,
    0xa831c66dUL, 0xb00327c8UL, 0xbf597fc7UL, 0xc6e00bf3UL, 0xd5a79147UL,
    0x06ca6351UL, 0x14292967UL, 0x27b70a85UL, 0x2e1b2138UL, 0x4d2c6dfcUL,
    0x53380d13UL, 0x650a7354UL, 0x766a0abbUL, 0x81c2c92eUL, 0x92722c85UL,
    0xa2bfe8a1UL, 0xa81a664bUL, 0xc24b8b70UL, 0xc76c51a3UL, 0xd192e819UL,
    0xd6990624UL, 0xf40e3585UL, 0x106aa070UL, 0x19a4c116UL, 0x1e376c08UL,
    0x2748774cUL, 0x34b0bcb5UL, 0x391c0cb3UL, 0x4ed8aa4aUL, 0x5b9cca4fUL,
    0x682e6ff3UL, 0x748f82eeUL, 0x78a5636fUL, 0x84c87814UL, 0x8cc70208UL,
    0x90befffaUL, 0xa4506cebUL, 0xbef9a3f7UL, 0xc67178f2UL
};

#define BLOCK_SIZE256          64

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  INTERNAL FUNCTIONS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Various logical functions
#define Ch( x, y, z )     (z ^ (x & (y ^ z)))
#define Maj( x, y, z )    (((x | y) & z) | (x & y))
#define S256( x, n )         ror((x),(n))
#define R256( x, n )         (((x)&0xFFFFFFFFUL)>>(n))
#define Sigma0_256( x )       (S256(x, 2) ^ S256(x, 13) ^ S256(x, 22))
#define Sigma1_256( x )       (S256(x, 6) ^ S256(x, 11) ^ S256(x, 25))
#define Gamma0_256( x )       (S256(x, 7) ^ S256(x, 18) ^ R256(x, 3))
#define Gamma1_256( x )       (S256(x, 17) ^ S256(x, 19) ^ R256(x, 10))

#define Sha256Round( a, b, c, d, e, f, g, h, i )       \
     t0 = h + Sigma1_256(e) + Ch(e, f, g) + K256[i] + W[i];   \
     t1 = Sigma0_256(a) + Maj(a, b, c);                    \
     d += t0;                                          \
     h  = t0 + t1;

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  TransformFunction
//
//  Compress 512-bits
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
static void TransformFunction_256(Sha256Context* Context, uint8_t const* Buffer)
{
    uint32_t    S[8];
    uint32_t    W[64];
    uint32_t    t0;
    uint32_t    t1;
    uint32_t    t;
    int         i;

    // Copy state into S
    for( i=0; i<8; i++ )
    {
        S[i] = Context->state[i];
    }

    // Copy the state into 512-bits into W[0..15]
    for( i=0; i<16; i++ )
    {
        LOAD32H( W[i], Buffer + (4*i) );
    }

    // Fill W[16..63]
    for( i=16; i<64; i++ )
    {
        W[i] = Gamma1_256( W[i-2]) + W[i-7] + Gamma0_256( W[i-15] ) + W[i-16];
    }

    // Compress
    for( i=0; i<64; i++ )
    {
        Sha256Round( S[0], S[1], S[2], S[3], S[4], S[5], S[6], S[7], i );
        t = S[7];
        S[7] = S[6];
        S[6] = S[5];
        S[5] = S[4];
        S[4] = S[3];
        S[3] = S[2];
        S[2] = S[1];
        S[1] = S[0];
        S[0] = t;
    }

    // Feedback
    for( i=0; i<8; i++ )
    {
        Context->state[i] = Context->state[i] + S[i];
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  PUBLIC FUNCTIONS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha256Initialise
//
//  Initialises a SHA256 Context. Use this to initialise/reset a context.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void Sha256Initialise(Sha256Context* Context)
{
    Context->curlen = 0;
    Context->length = 0;
    Context->state[0] = 0x6A09E667UL;
    Context->state[1] = 0xBB67AE85UL;
    Context->state[2] = 0x3C6EF372UL;
    Context->state[3] = 0xA54FF53AUL;
    Context->state[4] = 0x510E527FUL;
    Context->state[5] = 0x9B05688CUL;
    Context->state[6] = 0x1F83D9ABUL;
    Context->state[7] = 0x5BE0CD19UL;
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha256Update
//
//  Adds data to the SHA256 context. This will process the data and update the internal state of the context. Keep on
//  calling this function until all the data has been added. Then call Sha256Finalise to calculate the hash.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void Sha256Update(Sha256Context* Context, void const* Buffer, uint32_t BufferSize)
{
    uint32_t n;

    if( Context->curlen > sizeof(Context->buf) )
    {
       return;
    }

    while( BufferSize > 0 )
    {
        if( Context->curlen == 0 && BufferSize >= BLOCK_SIZE256 )
        {
           TransformFunction_256( Context, (uint8_t*)Buffer );
           Context->length += BLOCK_SIZE256 * 8;
           Buffer = (uint8_t*)Buffer + BLOCK_SIZE256;
           BufferSize -= BLOCK_SIZE256;
        }
        else
        {
           n = MIN( BufferSize, (BLOCK_SIZE256 - Context->curlen) );
           memcpy( Context->buf + Context->curlen, Buffer, (size_t)n );
           Context->curlen += n;
           Buffer = (uint8_t*)Buffer + n;
           BufferSize -= n;
           if( Context->curlen == BLOCK_SIZE256 )
           {
              TransformFunction_256( Context, Context->buf );
              Context->length += 8*BLOCK_SIZE256;
              Context->curlen = 0;
           }
       }
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha256Finalise
//
//  Performs the final calculation of the hash and returns the digest (32 byte buffer containing 256bit hash). After
//  calling this, Sha256Initialised must be used to reuse the context.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void Sha256Finalise(Sha256Context* Context, SHA256_HASH* Digest)
{
    int i;

    if( Context->curlen >= sizeof(Context->buf) )
    {
       return;
    }

    // Increase the length of the message
    Context->length += Context->curlen * 8;

    // Append the '1' bit
    Context->buf[Context->curlen++] = (uint8_t)0x80;

    // if the length is currently above 56 bytes we append zeros
    // then compress.  Then we can fall back to padding zeros and length
    // encoding like normal.
    if( Context->curlen > 56 )
    {
        while( Context->curlen < 64 )
        {
            Context->buf[Context->curlen++] = (uint8_t)0;
        }
        TransformFunction_256(Context, Context->buf);
        Context->curlen = 0;
    }

    // Pad up to 56 bytes of zeroes
    while( Context->curlen < 56 )
    {
        Context->buf[Context->curlen++] = (uint8_t)0;
    }

    // Store length
    STORE64H( Context->length, Context->buf+56 );
    TransformFunction_256( Context, Context->buf );

    // Copy output
    for( i=0; i<8; i++ )
    {
        STORE32H( Context->state[i], Digest->bytes+(4*i) );
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha256Calculate
//
//  Combines Sha256Initialise, Sha256Update, and Sha256Finalise into one function. Calculates the SHA256 hash of the
//  buffer.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void Sha256Calculate(void  const* Buffer, uint32_t BufferSize, SHA256_HASH* Digest)
{
    Sha256Context context;

    Sha256Initialise( &context );
    Sha256Update( &context, Buffer, BufferSize );
    Sha256Finalise( &context, Digest );
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  MACROS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

#define ROR64( value, bits ) (((value) >> (bits)) | ((value) << (64 - (bits))))

#define LOAD64H( x, y )                                                      \
   { x = (((uint64_t)((y)[0] & 255))<<56)|(((uint64_t)((y)[1] & 255))<<48) | \
         (((uint64_t)((y)[2] & 255))<<40)|(((uint64_t)((y)[3] & 255))<<32) | \
         (((uint64_t)((y)[4] & 255))<<24)|(((uint64_t)((y)[5] & 255))<<16) | \
         (((uint64_t)((y)[6] & 255))<<8)|(((uint64_t)((y)[7] & 255))); }

#define STORE64H( x, y )                                                                     \
   { (y)[0] = (uint8_t)(((x)>>56)&255); (y)[1] = (uint8_t)(((x)>>48)&255);     \
     (y)[2] = (uint8_t)(((x)>>40)&255); (y)[3] = (uint8_t)(((x)>>32)&255);     \
     (y)[4] = (uint8_t)(((x)>>24)&255); (y)[5] = (uint8_t)(((x)>>16)&255);     \
     (y)[6] = (uint8_t)(((x)>>8)&255); (y)[7] = (uint8_t)((x)&255); }

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  CONSTANTS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// The K array
static const uint64_t K512[80] = {
    0x428a2f98d728ae22ULL, 0x7137449123ef65cdULL, 0xb5c0fbcfec4d3b2fULL, 0xe9b5dba58189dbbcULL,
    0x3956c25bf348b538ULL, 0x59f111f1b605d019ULL, 0x923f82a4af194f9bULL, 0xab1c5ed5da6d8118ULL,
    0xd807aa98a3030242ULL, 0x12835b0145706fbeULL, 0x243185be4ee4b28cULL, 0x550c7dc3d5ffb4e2ULL,
    0x72be5d74f27b896fULL, 0x80deb1fe3b1696b1ULL, 0x9bdc06a725c71235ULL, 0xc19bf174cf692694ULL,
    0xe49b69c19ef14ad2ULL, 0xefbe4786384f25e3ULL, 0x0fc19dc68b8cd5b5ULL, 0x240ca1cc77ac9c65ULL,
    0x2de92c6f592b0275ULL, 0x4a7484aa6ea6e483ULL, 0x5cb0a9dcbd41fbd4ULL, 0x76f988da831153b5ULL,
    0x983e5152ee66dfabULL, 0xa831c66d2db43210ULL, 0xb00327c898fb213fULL, 0xbf597fc7beef0ee4ULL,
    0xc6e00bf33da88fc2ULL, 0xd5a79147930aa725ULL, 0x06ca6351e003826fULL, 0x142929670a0e6e70ULL,
    0x27b70a8546d22ffcULL, 0x2e1b21385c26c926ULL, 0x4d2c6dfc5ac42aedULL, 0x53380d139d95b3dfULL,
    0x650a73548baf63deULL, 0x766a0abb3c77b2a8ULL, 0x81c2c92e47edaee6ULL, 0x92722c851482353bULL,
    0xa2bfe8a14cf10364ULL, 0xa81a664bbc423001ULL, 0xc24b8b70d0f89791ULL, 0xc76c51a30654be30ULL,
    0xd192e819d6ef5218ULL, 0xd69906245565a910ULL, 0xf40e35855771202aULL, 0x106aa07032bbd1b8ULL,
    0x19a4c116b8d2d0c8ULL, 0x1e376c085141ab53ULL, 0x2748774cdf8eeb99ULL, 0x34b0bcb5e19b48a8ULL,
    0x391c0cb3c5c95a63ULL, 0x4ed8aa4ae3418acbULL, 0x5b9cca4f7763e373ULL, 0x682e6ff3d6b2b8a3ULL,
    0x748f82ee5defb2fcULL, 0x78a5636f43172f60ULL, 0x84c87814a1f0ab72ULL, 0x8cc702081a6439ecULL,
    0x90befffa23631e28ULL, 0xa4506cebde82bde9ULL, 0xbef9a3f7b2c67915ULL, 0xc67178f2e372532bULL,
    0xca273eceea26619cULL, 0xd186b8c721c0c207ULL, 0xeada7dd6cde0eb1eULL, 0xf57d4f7fee6ed178ULL,
    0x06f067aa72176fbaULL, 0x0a637dc5a2c898a6ULL, 0x113f9804bef90daeULL, 0x1b710b35131c471bULL,
    0x28db77f523047d84ULL, 0x32caab7b40c72493ULL, 0x3c9ebe0a15c9bebcULL, 0x431d67c49c100d4cULL,
    0x4cc5d4becb3e42b6ULL, 0x597f299cfc657e2aULL, 0x5fcb6fab3ad6faecULL, 0x6c44198c4a475817ULL
};

#define BLOCK_SIZE512          128

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  INTERNAL FUNCTIONS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Various logical functions
#define Ch( x, y, z )     (z ^ (x & (y ^ z)))
#define Maj(x, y, z )     (((x | y) & z) | (x & y))
#define S512( x, n )         ROR64( x, n )
#define R512( x, n )         (((x)&0xFFFFFFFFFFFFFFFFULL)>>((uint64_t)n))
#define Sigma0_512( x )       (S512(x, 28) ^ S512(x, 34) ^ S512(x, 39))
#define Sigma1_512( x )       (S512(x, 14) ^ S512(x, 18) ^ S512(x, 41))
#define Gamma0_512( x )       (S512(x, 1) ^ S512(x, 8) ^ R512(x, 7))
#define Gamma1_512( x )       (S512(x, 19) ^ S512(x, 61) ^ R512(x, 6))

#define Sha512Round( a, b, c, d, e, f, g, h, i )       \
     t0 = h + Sigma1_512(e) + Ch(e, f, g) + K512[i] + W[i];   \
     t1 = Sigma0_512(a) + Maj(a, b, c);                    \
     d += t0;                                          \
     h  = t0 + t1;

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  TransformFunction
//
//  Compress 1024-bits
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
static void TransformFunction_512(Sha512Context* Context, uint8_t const* Buffer) 
{
    uint64_t    S[8];
    uint64_t    W[80];
    uint64_t    t0;
    uint64_t    t1;
    int         i;

    // Copy state into S
    for( i=0; i<8; i++ )
    {
        S[i] = Context->state[i];
    }

    // Copy the state into 1024-bits into W[0..15]
    for( i=0; i<16; i++ )
    {
        LOAD64H(W[i], Buffer + (8*i));
    }

    // Fill W[16..79]
    for( i=16; i<80; i++ )
    {
        W[i] = Gamma1_512(W[i - 2]) + W[i - 7] + Gamma0_512(W[i - 15]) + W[i - 16];
    }

    // Compress
     for( i=0; i<80; i+=8 )
     {
         Sha512Round(S[0],S[1],S[2],S[3],S[4],S[5],S[6],S[7],i+0);
         Sha512Round(S[7],S[0],S[1],S[2],S[3],S[4],S[5],S[6],i+1);
         Sha512Round(S[6],S[7],S[0],S[1],S[2],S[3],S[4],S[5],i+2);
         Sha512Round(S[5],S[6],S[7],S[0],S[1],S[2],S[3],S[4],i+3);
         Sha512Round(S[4],S[5],S[6],S[7],S[0],S[1],S[2],S[3],i+4);
         Sha512Round(S[3],S[4],S[5],S[6],S[7],S[0],S[1],S[2],i+5);
         Sha512Round(S[2],S[3],S[4],S[5],S[6],S[7],S[0],S[1],i+6);
         Sha512Round(S[1],S[2],S[3],S[4],S[5],S[6],S[7],S[0],i+7);
     }

    // Feedback
    for( i=0; i<8; i++ )
    {
        Context->state[i] = Context->state[i] + S[i];
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  PUBLIC FUNCTIONS
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha512Initialise
//
//  Initialises a SHA512 Context. Use this to initialise/reset a context.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void Sha512Initialise(Sha512Context* Context) {
    Context->curlen = 0;
    Context->length = 0;
    Context->state[0] = 0x6a09e667f3bcc908ULL;
    Context->state[1] = 0xbb67ae8584caa73bULL;
    Context->state[2] = 0x3c6ef372fe94f82bULL;
    Context->state[3] = 0xa54ff53a5f1d36f1ULL;
    Context->state[4] = 0x510e527fade682d1ULL;
    Context->state[5] = 0x9b05688c2b3e6c1fULL;
    Context->state[6] = 0x1f83d9abfb41bd6bULL;
    Context->state[7] = 0x5be0cd19137e2179ULL;
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha512Update
//
//  Adds data to the SHA512 context. This will process the data and update the internal state of the context. Keep on
//  calling this function until all the data has been added. Then call Sha512Finalise to calculate the hash.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void
    Sha512Update
    (
        Sha512Context*      Context,        // [in out]
        void const*         Buffer,         // [in]
        uint32_t            BufferSize      // [in]
    )
{
    uint32_t    n;

    if( Context->curlen > sizeof(Context->buf) )
    {
       return;
    }

    while( BufferSize > 0 )
    {
        if( Context->curlen == 0 && BufferSize >= BLOCK_SIZE512 )
        {
           TransformFunction_512( Context, (uint8_t *)Buffer );
           Context->length += BLOCK_SIZE512 * 8;
           Buffer = (uint8_t*)Buffer + BLOCK_SIZE512;
           BufferSize -= BLOCK_SIZE512;
        }
        else
        {
           n = MIN( BufferSize, (BLOCK_SIZE512 - Context->curlen) );
           memcpy( Context->buf + Context->curlen, Buffer, (size_t)n );
           Context->curlen += n;
           Buffer = (uint8_t*)Buffer + n;
           BufferSize -= n;
           if( Context->curlen == BLOCK_SIZE512 )
           {
              TransformFunction_512( Context, Context->buf );
              Context->length += 8*BLOCK_SIZE512;
              Context->curlen = 0;
           }
       }
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha512Finalise
//
//  Performs the final calculation of the hash and returns the digest (64 byte buffer containing 512bit hash). After
//  calling this, Sha512Initialised must be used to reuse the context.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void
    Sha512Finalise
    (
        Sha512Context*      Context,        // [in out]
        SHA512_HASH*        Digest          // [out]
    )
{
    int i;

    if( Context->curlen >= sizeof(Context->buf) )
    {
       return;
    }

    // Increase the length of the message
    Context->length += Context->curlen * 8ULL;

    // Append the '1' bit
    Context->buf[Context->curlen++] = (uint8_t)0x80;

    // If the length is currently above 112 bytes we append zeros
    // then compress.  Then we can fall back to padding zeros and length
    // encoding like normal.
    if( Context->curlen > 112 )
    {
        while( Context->curlen < 128 )
        {
            Context->buf[Context->curlen++] = (uint8_t)0;
        }
        TransformFunction_512( Context, Context->buf );
        Context->curlen = 0;
    }

    // Pad up to 120 bytes of zeroes
    // note: that from 112 to 120 is the 64 MSB of the length.  We assume that you won't hash
    // > 2^64 bits of data... :-)
    while( Context->curlen < 120 )
    {
        Context->buf[Context->curlen++] = (uint8_t)0;
    }

    // Store length
    STORE64H( Context->length, Context->buf+120 );
    TransformFunction_512( Context, Context->buf );

    // Copy output
    for( i=0; i<8; i++ )
    {
        STORE64H( Context->state[i], Digest->bytes+(8*i) );
    }
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//  Sha512Calculate
//
//  Combines Sha512Initialise, Sha512Update, and Sha512Finalise into one function. Calculates the SHA512 hash of the
//  buffer.
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
void
    Sha512Calculate
    (
        void  const*        Buffer,         // [in]
        uint32_t            BufferSize,     // [in]
        SHA512_HASH*        Digest          // [in]
    )
{
    Sha512Context context;

    Sha512Initialise( &context );
    Sha512Update( &context, Buffer, BufferSize );
    Sha512Finalise( &context, Digest );
}