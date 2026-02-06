#ifndef DDP_GC_H
#define DDP_GC_H

#include "ddptypes.h"

#ifdef DDP_DEBUG
#define DDP_GC_STRESS 0
#else
#define DDP_GC_STRESS 0
#endif // DDP_DEBUG

void ddp_init_gc(void);
void ddp_gc(void);
void ddp_free_gc(void);

static_assert(sizeof(void *) == 8, "Expecting pointer size to be 8 byte");

// type metadata for the GC
// TODO: account for arrays
typedef struct GCTypeMeta {
  ddpvtable *vtable;
  // bit mask which quads are themselves roots -> limits
  // object size to (8 * 64) byte;
  uint64_t ptrmask;
  ddpint arrlen; // if arrlen > 0 -> the Object is an array of length arrlen
} GCTypeMeta;

void ddp_register_gc_root(void *root);
void ddp_free_gc_ref(void *ref);
void *ddp_allocate_gc_ref(ddpvtable *vtable);

#endif // DDP_GC_H
