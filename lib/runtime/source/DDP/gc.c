#include "DDP/gc.h"
#include "DDP/common.h"
#include "DDP/ddpmemory.h"
#include "DDP/ddpwindows.h"
#include "DDP/debug.h"
#include <stddef.h>
#include <unistd.h>

#define UNUSED __attribute__((unused))

/*
 * Implements a Parser for the stack map.
 * See llvm-project/llvm/include/Object/StackMapParser.h
*/

// StackMap offsets
#define HeaderOffset 0
#define NumFunctionsOffset (HeaderOffset + sizeof(uint32_t))
#define NumConstantsOffset (NumFunctionsOffset + sizeof(uint32_t))
#define NumRecordsOffset (NumConstantsOffset + sizeof(uint32_t))
#define FunctionListOffset (NumRecordsOffset + sizeof(uint32_t))
#define FunctionSize (3 * sizeof(uint64_t))
#define ConstantSize (sizeof(uint64_t))

// StackMapRecord offsets
#define PatchpointIDOffset 0
#define InstructionOffsetOffset (PatchpointIDOffset + sizeof(uint64_t))
#define NumLocationsOffset (InstructionOffsetOffset + sizeof(uint32_t) + sizeof(uint16_t))
#define LocationListOffset (NumLocationsOffset + sizeof(uint16_t))
#define LocationSize (sizeof(uint64_t) + sizeof(uint32_t))
#define LiveOutSize (sizeof(uint32_t))

// StackMapLiveout offsets
#define LiveoutDwarfRegNumOffset 0
#define LiveoutSizeOffset (LiveoutDwarfRegNumOffset + sizeof(uint16_t) + sizeof(uint8_t))
#define LiveOutAccessorSize sizeof(uint32_t)

// StackMapLocation offsets
#define KindOffset 0
#define LocationSizeOffset (KindOffset + sizeof(uint16_t))
#define LocationDwarfRegNumOffset (LocationSizeOffset + sizeof(uint16_t))
#define SmallConstantOffset (LocationDwarfRegNumOffset + sizeof(uint32_t))
#define LocationAccessorSize (sizeof(uint64_t) + sizeof(uint32_t))

#define READ_AS(p, T) (*(T *)(p))

#define LOC_REGISTER 0x1
#define LOC_DIRECT 0x2
#define LOC_INDIRECT 0x3
#define LOC_CONSTANT 0x4
#define LOC_CONST_INDEX 0x5

typedef struct StackMapHeader {
	uint8_t version; // (current version is 3)
	uint8_t _reserved1;
	uint16_t _reserved2;
} StackMapHeader;

typedef struct StackSizeRecord {
	uint64_t functionAddress;
	uint64_t stackSize; // (or UINT64_MAX if not statically known)
	uint64_t recordCount;
} StackSizeRecord;

typedef struct Constant {
	uint64_t largeConstant;
} Constant;

typedef struct Location {
	uint8_t type; // Register | Direct | Indirect | Constant | ConstantIndex
	uint8_t _reserved1;
	uint16_t locationSize;
	uint16_t dwarfRegNum;
	uint16_t _reserved2;
	int32_t offsetOrSmallConstant;
} Location;

typedef struct LiveOut {
	uint16_t dwarfRegNum;
	uint8_t _reserved;
	uint8_t sizeInBytes;
} LiveOut;

// typedef struct StackMapRecord {
// 	uint64_t patchPointID;
// 	uint32_t instructionOffset;
// 	uint16_t _reserved;
// 	uint16_t numLocations;
// 	Location *locations; // [NumLocations]
// 	uint32_t _padding1;	 // (only if required to align to 8 byte)
// 	uint16_t _padding2;
// 	uint16_t numLiveOuts;
// 	LiveOut *liveOuts;	// [NumLiveOuts]
// 	uint32_t _padding3; // (only if required to align to 8 byte)
// } StackMapRecord;

typedef const uint8_t *Accessor;
typedef Accessor LocationAccessor;
typedef Accessor LiveOutAccessor;
typedef Accessor RecordAccessor;

static uint16_t location_size(LocationAccessor location) {
	return READ_AS(&location[LocationSizeOffset], uint16_t);
}

static uint8_t UNUSED location_kind(LocationAccessor location) {
	return READ_AS(&location[KindOffset], uint8_t);
}

static uint16_t UNUSED location_dwarf_regnum(LocationAccessor location) {
	return READ_AS(&location[LocationDwarfRegNumOffset], uint16_t);
}

static uint32_t UNUSED location_small_constant(LocationAccessor location) {
	return READ_AS(&location[SmallConstantOffset], uint32_t);
}

static uint32_t UNUSED location_constant_index(LocationAccessor location) {
	return READ_AS(&location[SmallConstantOffset], uint32_t);
}

static int32_t UNUSED location_offset(LocationAccessor location) {
	return READ_AS(&location[SmallConstantOffset], int32_t);
}

static LocationAccessor UNUSED next_location_accessor(LocationAccessor location) {
	return &location[location_size(location)];
}

static unsigned liveout_size(LiveOutAccessor liveout) {
	return (unsigned)READ_AS(&liveout[LiveoutSizeOffset], uint8_t);
}

static uint16_t UNUSED liveout_dwarf_regnum(LiveOutAccessor liveout) {
	return READ_AS(&liveout[LiveoutDwarfRegNumOffset], uint16_t);
}

static LiveOutAccessor UNUSED next_liveout_accessor(LiveOutAccessor location) {
	return &location[liveout_size(location)];
}

static uint16_t record_num_locations(RecordAccessor record) {
	return READ_AS(&record[NumLocationsOffset], uint16_t);
}

static uint16_t record_num_liveouts_offset(RecordAccessor record) {
	const unsigned LocOffset =
		((LocationListOffset + LocationSize * record_num_locations(record)) + 7) & ~0x7;
	return (LocOffset + sizeof(uint16_t));
}

static uint16_t record_num_liveouts(RecordAccessor record) {
	return READ_AS(&record[record_num_liveouts_offset(record)], uint16_t);
}

static uint64_t UNUSED record_patchpoint_id(RecordAccessor record) {
	return READ_AS(&record[PatchpointIDOffset], uint64_t);
}

static uint32_t UNUSED record_instruction_offset(RecordAccessor record) {
	return READ_AS(&record[InstructionOffsetOffset], uint32_t);
}

static unsigned record_size(RecordAccessor record) {
	const unsigned numLiveOutsOffset = record_num_liveouts_offset(record);
	const unsigned recordSize =
		numLiveOutsOffset + sizeof(uint16_t) +
		record_num_liveouts(record) * LiveOutSize;
	return (recordSize + 7) & ~0x7;
}

static RecordAccessor next_record_accessor(RecordAccessor record) {
	return &record[record_size(record)];
}

typedef struct StackMap {
	const uint8_t *base;
	StackMapHeader header;
	uint32_t numFunctions;
	uint32_t numConstants;
	uint32_t numRecords;
	const StackSizeRecord *functions; // [NumFunctions]
	const Constant *constants;		  // [NumConstants]
	RecordAccessor records_base;	  // [NumRecords]
	unsigned *record_offsets;		  // array of offsets
} StackMap;

static void read_stack_map(StackMap *stackMap, const uint8_t *section) {
	stackMap->base = section;
	stackMap->header = READ_AS(section, StackMapHeader);
	if (stackMap->header.version != 3) {
		ddp_runtime_error(3, "Unable to initialize GC: StackMap Version (%hhu) != 3\n", stackMap->header.version);
	}

	if ((((intptr_t)section) % 8) != 0) {
		ddp_runtime_error(3, "Stack Map (%p) is not 8 byte aligned\n", (void *)section);
	}

	stackMap->numFunctions = READ_AS(&section[NumFunctionsOffset], uint32_t);
	stackMap->numConstants = READ_AS(&section[NumConstantsOffset], uint32_t);
	stackMap->numRecords = READ_AS(&section[NumRecordsOffset], uint32_t);

	DDP_DBGLOG("StackMap: %u %u %u", stackMap->numFunctions, stackMap->numConstants, stackMap->numRecords);

	const int ConstantsListOffset = FunctionListOffset + stackMap->numFunctions * FunctionSize;
	unsigned currentRecordOffset = ConstantsListOffset + stackMap->numConstants * ConstantSize;

	stackMap->functions = (StackSizeRecord *)&section[FunctionListOffset];
	stackMap->constants = (Constant *)&section[ConstantsListOffset];
	stackMap->records_base = (RecordAccessor)&section[currentRecordOffset];
	stackMap->record_offsets = DDP_ALLOCATE(unsigned, stackMap->numRecords);

	for (unsigned i = 0; i < stackMap->numRecords; i++) {
		stackMap->record_offsets[i] = currentRecordOffset;
		const unsigned size = record_size(&section[currentRecordOffset]);
		currentRecordOffset += size;
	}

	DDP_DBGLOG("Done reading Stack Map");
}

static void free_stack_map(StackMap *stackMap) {
	DDP_FREE_ARRAY(unsigned, (void *)stackMap->record_offsets, stackMap->numRecords);
}

static RecordAccessor UNUSED get_record(StackMap *stackMap, unsigned index) {
	unsigned offset = stackMap->record_offsets[index];
	return &stackMap->base[offset];
}

static void UNUSED dump_stackmap(StackMap *stackMap) {
	DDP_DBGLOG("Functions:\n\n");
	for (const StackSizeRecord *function = stackMap->functions; function < &stackMap->functions[stackMap->numFunctions]; function++) {
		DDP_DBGLOG("Function: %llu %llu %llu\n", function->functionAddress, function->stackSize, function->recordCount);
	}

	DDP_DBGLOG("\nConstants:\n\n");
	for (const Constant *constant = stackMap->constants; constant < &stackMap->constants[stackMap->numConstants]; constant++) {
		DDP_DBGLOG("Constant: %llu\n", constant->largeConstant);
	}

	DDP_DBGLOG("\nRecords:\n\n");
	for (RecordAccessor record = stackMap->records_base; record != next_record_accessor(get_record(stackMap, stackMap->numRecords - 1)); record = next_record_accessor(record)) {
		DDP_DBGLOG("Record Value: %llu %u %hu\n", record_patchpoint_id(record), record_instruction_offset(record), record_num_locations(record));
	}
}

// GC

// TODO: incremental GC (tri-color marking)
// TODO: mark intermediate values as roots

static size_t UNUSED get_page_size(void) {
#ifdef DDPOS_WINDOWS
	SYSTEM_INFO sysInfo;
	GetSystemInfo(&sysInfo);

	return sysInfo.dwPageSize;
#else
#endif
}

static bool type_meta_equal(GCTypeMeta a, GCTypeMeta b) {
	return a.vtable == b.vtable && a.ptrmask == b.ptrmask;
}

static int bitmap_find_first_zero(uint8_t *bitmap, size_t num_bits) {
	size_t num_bytes = (num_bits + 7) / 8;
	for (size_t byte = 0; byte < num_bytes; byte++) {
		if (bitmap[byte] != 0xFF) {
			// There is at least one zero bit here
			for (int bit = 0; bit < 8; bit++) {
				size_t bit_index = byte * 8 + bit;
				if (bit_index >= num_bits) {
					return -1;
				}

				if ((bitmap[byte] & (1u << bit)) == 0) {
					return (ssize_t)bit_index;
				}
			}
		}
	}
	return -1; // all bits are 1
}

static void bitmap_set_bit(uint8_t *bitmap, size_t index) {
	bitmap[index / 8] |= 1 << (index % 8);
}

static void bitmap_clear_bit(uint8_t *bitmap, size_t index) {
	bitmap[index / 8] &= ~(1 << (index % 8));
}

static int UNUSED bitmap_get_bit(uint8_t *bitmap, size_t index) {
	return bitmap[index / 8] & (1 << (index % 8));
}

// sets the bit at index and at index+1 to the bits specified in value&0x3
static void bitmap_set_two_bits(uint8_t *bitmap, size_t index, uint8_t value) {
	bitmap[index / 8] |= (value & 0x3) << (index % 8);
}
//
// sets the bit at index and at index+1 to the bits specified in value&0x3
static void bitmap_clear_two_bits(uint8_t *bitmap, size_t index) {
	bitmap[index / 8] &= ~(0x3 << (index % 8));
}

// gets the bit at index and at index+1
static int UNUSED bitmap_get_two_bits(uint8_t *bitmap, size_t index) {
	return bitmap[index / 8] & (0x3 << (index % 8));
}

// round up positive number to nearest multiple
static int round_up_to_multiple(size_t numToRound, size_t multiple) {
	if (multiple == 0) {
		return numToRound;
	}

	size_t remainder = numToRound % multiple;
	if (remainder == 0) {
		return numToRound;
	}

	return numToRound + multiple - remainder;
}

typedef enum Color {
	WHITE = 0,
	GREY = 1,
	BLACK = 2
} Color;

typedef struct GCSpan {
	GCTypeMeta objInfo;

	void *data;
	size_t allocatedSize; // bytes actually allocated, including potential waste
	uint32_t objSize;	  // size of a single object (arrays count as single objects)
	uint16_t numObjs;

	uint8_t *freeBits;
	uint8_t *markBits;

	struct GCSpan *next;
} GCSpan;

typedef struct GC {
	StackMap stackMap;

	void ***global_roots;
	unsigned len_global_roots;
	unsigned cap_global_roots;

	GCSpan *spanHead;
	GCSpan *spanTail;

	size_t PAGE_SIZE;

	bool collecting;
} GC;

static GC gc;

static void *osAlloc(void *hint UNUSED, size_t nbytes) {
#ifdef DDPOS_WINDOWS
	// TODO: use hint
	void *result = VirtualAlloc(NULL, nbytes, MEM_COMMIT, PAGE_READWRITE);
#else
	// TODO: use mmap
#endif
	DDP_DBGLOG("Allocated %p from OS", result);
	return result;
}

static void osFree(void *p, size_t nbytes) {
#ifdef DDPOS_WINDOWS
	VirtualFree(p, nbytes, MEM_RELEASE);
#else
	// TODO: use munmap
#endif
}

static size_t calculate_span_size(size_t objSize) {
	// const size_t perPage = gc.PAGE_SIZE / objSize;

	// large objects
	if (objSize > gc.PAGE_SIZE) {
		return round_up_to_multiple(objSize, gc.PAGE_SIZE);
	}

	return gc.PAGE_SIZE; // TODO: calculate different size classes
}

#define NUM_FREE_BYTES(numObjs) ((size_t)ceil((double)(numObjs) / 8.0))
#define NUM_MARK_BYTES(numObjs) ((size_t)ceil((double)(numObjs) / 4.0)) // 2 bits because we have 3 colors

// TODO: use size classes
static GCSpan *UNUSED allocate_span(GCTypeMeta objInfo) {
	DDP_DBGLOG("Allocating new span");

	GCSpan *newSpan = DDP_ALLOCATE(GCSpan, 1);
	newSpan->objInfo = objInfo;
	newSpan->objSize = objInfo.vtable->type_size * (objInfo.arrlen > 0 ? objInfo.arrlen : 1);
	newSpan->allocatedSize = calculate_span_size(newSpan->objSize);
	newSpan->data = osAlloc(gc.spanTail == NULL ? NULL : &((uint8_t *)gc.spanTail->data)[newSpan->allocatedSize], newSpan->allocatedSize);
	newSpan->numObjs = newSpan->allocatedSize / newSpan->objSize;

	const size_t freeBytes = NUM_FREE_BYTES(newSpan->numObjs);
	const size_t markBytes = NUM_MARK_BYTES(newSpan->numObjs);
	newSpan->freeBits = DDP_ALLOCATE(uint8_t, freeBytes);
	memset(newSpan->freeBits, 0, freeBytes);
	newSpan->markBits = DDP_ALLOCATE(uint8_t, markBytes);
	memset(newSpan->markBits, 0, markBytes);
	newSpan->next = NULL;

	if (gc.spanHead == NULL) {
		gc.spanTail = gc.spanHead = newSpan;
	} else {
		gc.spanTail->next = newSpan;
	}

	DDP_DBGLOG("Allocated new span");

	return newSpan;
}

static void UNUSED free_span(GCSpan *span, GCSpan *prev) {
	GCSpan *next = span->next;
	if (prev) {
		prev->next = next;
	}
	if (gc.spanHead == span) {
		gc.spanHead = next;
	}

	const size_t freeBytes = NUM_FREE_BYTES(span->numObjs);
	const size_t markBytes = NUM_MARK_BYTES(span->numObjs);
	DDP_FREE_ARRAY(uint8_t, span->freeBits, freeBytes);
	DDP_FREE_ARRAY(uint8_t, span->markBits, markBytes);
	osFree(span->data, span->allocatedSize);
	DDP_FREE(GCSpan, span);
}

static void clear_free_spans(void) {
	GCSpan *prev = NULL;
	for (GCSpan *span = gc.spanHead; span != NULL;) {
		if (span->freeBits[0] == 0 && memcmp(span->freeBits, span->freeBits + 1, NUM_FREE_BYTES(span->numObjs) - 1) == 0) {
			GCSpan *next = span->next;
			free_span(span, prev);
			span = next;
		} else {
			prev = span;
			span = span->next;
		}
	}
}

// TODO: use a binary search lookup table
static UNUSED GCSpan *get_span_for_pointer(void *p) {
	for (GCSpan *span = gc.spanHead; span != NULL; span = span->next) {
		DDP_DBGLOG("Checking span %p for %p", span->data, p);
		// maybe change this. see https://devblogs.microsoft.com/oldnewthing/20170927-00/?p=97095
		if ((uint8_t *)p >= (uint8_t *)span->data && (uint8_t *)p < ((uint8_t *)span->data + span->allocatedSize)) {
			return span;
		}
	}
	DDP_DBGLOG("No span found");
	return NULL;
}

static void mark_node(void *ref, Color color) {
	GCSpan *span = get_span_for_pointer(ref);
	if (span == NULL) {
		return;
	}

	DDP_DBGLOG("marking node %p with color %d", ref, color);

	unsigned index = (((((uint8_t *)ref) - ((uint8_t *)span->data)) / span->objSize)) * 2 + 1; // * 2 + 1 to account for two bits per object
	if (color == 0) {
		bitmap_clear_two_bits(span->markBits, index);
	} else {
		bitmap_set_two_bits(span->markBits, index, color);
	}
}

static GCSpan *find_or_allocate_span_for_object(GCTypeMeta objInfo, void **space_slot) {
	for (GCSpan *span = gc.spanHead; span != NULL; span = span->next) {
		if (type_meta_equal(span->objInfo, objInfo)) {
			int free_slot = bitmap_find_first_zero(span->freeBits, span->numObjs);
			DDP_DBGLOG("Found free slot: %d", free_slot);
			if (free_slot >= 0) {
				*space_slot = (void *)(((uint8_t *)span->data) + free_slot * span->objSize);
				return span;
			}
		}
	}

	GCSpan *newSpan = allocate_span(objInfo);
	*space_slot = newSpan->data;
	return newSpan;
}

void ddp_register_gc_root(void **root) {
	DDP_DBGLOG("Registering root %p", root);

	if (gc.len_global_roots == gc.cap_global_roots) {
		gc.cap_global_roots += 8;
		bool c = gc.collecting;
		gc.collecting = true;
		gc.global_roots = DDP_GROW_ARRAY(void **, gc.global_roots, gc.len_global_roots, gc.cap_global_roots);
		gc.collecting = c;
	}

	gc.global_roots[gc.len_global_roots++] = root;
}

void ddp_free_gc_ref(void *ref UNUSED) {
	DDP_DBGLOG("Freeing ref: %p, Span: %p", ref, get_span_for_pointer(ref));
}

void *ddp_allocate_gc_ref(ddpvtable *vtable) {
	DDP_DBGLOG("Allocating GC ref from vtable: %p", vtable);
	GCTypeMeta objInfo = {.vtable = vtable, .ptrmask = 0}; // TODO: get ptrmask
	void *space_slot = NULL;
	GCSpan *span = find_or_allocate_span_for_object(objInfo, &space_slot);

	unsigned index = (((uint8_t *)space_slot) - ((uint8_t *)span->data)) / span->objSize;
	bitmap_set_bit(span->freeBits, index);

	DDP_DBGLOG("allocated ref: %p", space_slot);
	return space_slot;
}

// pointer to the .llvm_stackmaps section
// defined by the compiler in the main object file, because the section itself is a local symbol
extern const uint8_t *__LLVM_StackMaps_External;

void ddp_init_gc(void) {
	DDP_DBGLOG("initializing gc");

	read_stack_map(&gc.stackMap, __LLVM_StackMaps_External);
	// dump_stackmap(&stackMap);
	gc.global_roots = NULL;
	gc.len_global_roots = 0;
	gc.cap_global_roots = 0;

	gc.spanHead = NULL;
	gc.spanTail = NULL;

	gc.PAGE_SIZE = get_page_size();

	gc.collecting = false;

	DDP_DBGLOG("done initializing gc");
}

void ddp_gc(void) {
	if (gc.collecting) {
		return;
	}
	gc.collecting = true;

	DDP_DBGLOG("GC start");

	// mark global roots
	for (void ***root = gc.global_roots; root != &gc.global_roots[gc.len_global_roots]; root++) {
		if (*root != NULL) {
			DDP_DBGLOG("Root %p in use (Ref value: %p) (Span: %p)", *root, **root, get_span_for_pointer(**root));
		} else {
			DDP_DBGLOG("Root %p not in use", *root);
		}

		// ignore null refs
		if (*root == NULL) {
			continue;
		}

		mark_node(**root, GREY);

		// TODO: traverse object using ptrmask

		mark_node(**root, BLACK);
	}

	// free unmarked objects
	for (GCSpan *span = gc.spanHead; span != NULL; span = span->next) {
		free_func_ptr free_func = span->objInfo.vtable->free_func;

		// loop over and free every single white object
		for (unsigned i = 0; i < NUM_FREE_BYTES(span->numObjs); i++) {
			uint8_t byte = span->freeBits[i];
			if (byte == 0) {
				continue;
			}

			for (int oneIndex = 0; oneIndex < 8; oneIndex++) {
				if ((byte & (1 << oneIndex)) == 0) {
					break;
				}

				unsigned index = oneIndex + i * 8;

				DDP_DBGLOG("color of index %u is %d", index, bitmap_get_two_bits(span->markBits, index));
				if (bitmap_get_two_bits(span->markBits, index) != WHITE) {
					continue;
				}

				DDP_DBGLOG("freeing gc ref %p", (void *)&((uint8_t *)span->data)[index * span->objSize]);

				bitmap_clear_bit(span->freeBits, index);
				byte &= ~(1 << (index % 8));
				DDP_DBGLOG("i: %d, bitmask after clear: %hhx", i, byte);

				// free the object
				if (free_func != NULL) {
					free_func((void *)&((uint8_t *)span->data)[index * span->objSize]);
				}
			}
		}

		// mark all objects white before the next collection
		memset(span->markBits, WHITE, NUM_MARK_BYTES(span->numObjs));
	}

	clear_free_spans();

	gc.collecting = false;
	DDP_DBGLOG("GC end");
}

void ddp_free_gc(void) {
	DDP_FREE_ARRAY(void *, gc.global_roots, gc.cap_global_roots);
	gc.global_roots = NULL;
	gc.len_global_roots = 0;
	gc.cap_global_roots = 0;
	// gc one last time with cleaned roots
	DDP_DBGLOG("last gc");
	ddp_gc();

	free_stack_map(&gc.stackMap);
}
