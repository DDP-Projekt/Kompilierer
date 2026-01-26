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
typedef struct GCTypeMeta {
  ddpvtable *vtable;
  uint64_t ptrmask; // bit mask which quads are themselves roots -> limits object size to 8 * 64 byte;
} GCTypeMeta;

void ddp_register_gc_root(void *root);

#endif // DDP_GC_H
