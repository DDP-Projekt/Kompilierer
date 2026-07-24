#include "DDP/gc.h"
#include "DDP/common.h"
#include "DDP/ddpmemory.h"
#include "DDP/ddpwindows.h"
#include "DDP/debug.h"
#include <stdalign.h>
#include <stddef.h>
#include <string.h>
#include <unistd.h>

#ifdef DDPOS_LINUX
#include <sys/mman.h>
#endif

#define UNW_LOCAL_ONLY
#include "libunwind.h"

#define UNUSED __attribute__((unused))
#define ALWAYS_INLINE __attribute__((always_inline))

static void *ddp_reallocate_no_gc(void *pointer, size_t oldSize, size_t newSize);

#define DDP_ALLOCATE_NO_GC(type, count) \
	(type *)ddp_reallocate_no_gc(NULL, 0, sizeof(type) * (count))

// helper macro to free any type (not arrays though)
#define DDP_FREE_NO_GC(type, pointer) ddp_reallocate_no_gc(pointer, sizeof(type), 0)

// helper macro to expand the capacity of an array
#define DDP_GROW_ARRAY_NO_GC(type, pointer, oldCount, newCount)      \
	(type *)ddp_reallocate_no_gc(pointer, sizeof(type) * (oldCount), \
								 sizeof(type) * (newCount))

// helper to free a whole array
#define DDP_FREE_ARRAY_NO_GC(type, pointer, oldCount) \
	ddp_reallocate_no_gc(pointer, sizeof(type) * (oldCount), 0)

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

static uint8_t location_kind(LocationAccessor location) {
	return READ_AS(&location[KindOffset], uint8_t);
}

static uint16_t location_dwarf_regnum(LocationAccessor location) {
	return READ_AS(&location[LocationDwarfRegNumOffset], uint16_t);
}

static uint32_t UNUSED location_small_constant(LocationAccessor location) {
	return READ_AS(&location[SmallConstantOffset], uint32_t);
}

static uint32_t UNUSED location_constant_index(LocationAccessor location) {
	return READ_AS(&location[SmallConstantOffset], uint32_t);
}

static int32_t location_offset(LocationAccessor location) {
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

static uint32_t record_instruction_offset(RecordAccessor record) {
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
	stackMap->record_offsets = DDP_ALLOCATE_NO_GC(unsigned, stackMap->numRecords);

	for (unsigned i = 0; i < stackMap->numRecords; i++) {
		stackMap->record_offsets[i] = currentRecordOffset;
		const unsigned size = record_size(&section[currentRecordOffset]);
		currentRecordOffset += size;
	}

	DDP_DBGLOG("Done reading Stack Map");
}

static void free_stack_map(StackMap *stackMap) {
	DDP_FREE_ARRAY_NO_GC(unsigned, (void *)stackMap->record_offsets, stackMap->numRecords);
}

static RecordAccessor get_record(StackMap *stackMap, unsigned index) {
	unsigned offset = stackMap->record_offsets[index];
	return &stackMap->base[offset];
}

static LocationAccessor get_location(RecordAccessor record, unsigned index) {
	unsigned offset = LocationListOffset + index * LocationSize;
	return (LocationAccessor)(record + offset);
}

static const Constant *get_constant(StackMap *stackMap, unsigned index) {
	return stackMap->constants + index * ConstantSize;
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

static const StackSizeRecord *find_function_for_pc(StackMap *stackMap, unw_word_t pc) {
	for (unsigned i = 0; i < stackMap->numFunctions; i++) {
		uint64_t start = stackMap->functions[i].functionAddress;
		uint64_t end =
			(i + 1 < stackMap->numFunctions) ? stackMap->functions[i + 1].functionAddress : UINT64_MAX; // last function
		if (pc >= start && pc < end) {
			return &stackMap->functions[i];
		}
	}
	return NULL;
}

RecordAccessor first_record_for_function(StackMap *stackMap, const StackSizeRecord *f) {
	unsigned index = 0;
	for (const StackSizeRecord *function = stackMap->functions; function < f; function++) {
		index += function->recordCount;
	}

	return get_record(stackMap, index);
}

static RecordAccessor find_stackmap_record(StackMap *stackMap, unw_word_t pc) {
	const StackSizeRecord *function = find_function_for_pc(stackMap, pc);
	if (function == NULL) {
		return NULL;
	}

	const uint64_t offset = pc - function->functionAddress;

	RecordAccessor record = first_record_for_function(stackMap, function);

	for (unsigned i = 0; i < function->recordCount; i++) {
		uint32_t instOff = record_instruction_offset(record);

		if (offset == instOff ||
			offset == instOff + 1) { // tolerate PC-after-instruction
			return record;
		}

		record = next_record_accessor(record);
	}

	return NULL;
}

// GC

// TODO: incremental GC (tri-color marking)
// TODO: mark intermediate values as roots

static size_t get_page_size(void) {
#ifdef DDPOS_WINDOWS
	SYSTEM_INFO sysInfo;
	GetSystemInfo(&sysInfo);

	return sysInfo.dwPageSize;
#else
	long pagesize = sysconf(_SC_PAGESIZE);
	if (pagesize <= 0) {
		return (1 << 12); // Fallback to 4kb
	}
	return pagesize;
#endif
}

// the GC does not care wether two types are actually the same, just wether they can be stored on the same span
// this only really matters for typedef-references if they are cast from/to their underlying type
// but it still affects the span allocation of other types
static bool vtable_gc_equal(ddpvtable *a, ddpvtable *b) {
	return a == b ||
		   (a->type_size == b->type_size &&
			a->free_func == b->free_func &&
			memcmp(a->ptrmask, b->ptrmask, sizeof(a->ptrmask)) == 0);
}

static bool type_meta_equal(GCTypeMeta a, GCTypeMeta b) {
	return a.arrlen == b.arrlen && vtable_gc_equal(a.vtable, b.vtable);
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
	return (bitmap[index / 8] & (1 << (index % 8))) >> (index % 8);
}

// sets the two bits at index to the bits specified in value & 0x3
// correct index calculation for a two-bit-pair bitmap is done here
static void bitmap_set_two_bits(uint8_t *bitmap, size_t index, uint8_t value) {
	bitmap[index / 8] |= (value & 0x3) << ((index * 2) % 8);
}

// sets the two bits at index to the bits specified in value & 0x3
// correct index calculation for a two-bit-pair bitmap is done here
static void bitmap_clear_two_bits(uint8_t *bitmap, size_t index) {
	bitmap[index / 8] &= ~(0x3 << ((index * 2) % 8));
}

// gets the bits at index
// correct index calculation for a two-bit-pair bitmap is done here
static int bitmap_get_two_bits(uint8_t *bitmap, size_t index) {
	return (bitmap[index / 8] & (uint8_t)(0x3 << ((index * 2) % 8))) >> ((index * 2) % 8);
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

#if defined(__x86_64__) || defined(_M_X64)
#define TAG_BIT ((uintptr_t)1 << 63)
#elif defined(__aarch64__) || defined(_M_ARM64)
#error "TBI needs to be enabled before using high-bit pointer tagging on ARM"
#else
#error "Unsupported architecture for high-bit pointer tagging"
#endif

static inline void *with_tag(void *ptr, int flag) {
	uintptr_t addr = (uintptr_t)ptr;
	return flag ? (void *)(addr | TAG_BIT) : (void *)(addr & ~TAG_BIT);
}

static inline int get_tag(const void *ptr) {
	return (int)(((uintptr_t)ptr & TAG_BIT) != 0);
}

static inline void *without_tag(const void *ptr) {
	return (void *)((uintptr_t)ptr & ~TAG_BIT);
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

#ifdef DDPOS_WINDOWS
// wrapper to get an error message for GetLastError()
// expects fmt to be of format "<message>%s"
static void runtime_error_getlasterror(int exit_code, const char *fmt) {
	char error_buffer[1024];
	DWORD error_code = GetLastError();
	if (!FormatMessageA(FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
						NULL, error_code, MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), error_buffer, sizeof(error_buffer), NULL)) {
		sprintf(error_buffer, "WinAPI Error Code %d (FormatMessageA failed with code %d)", error_code, GetLastError());
	}
	ddp_runtime_error(exit_code, fmt, error_buffer);
}
#endif // DDPOS_WINDOWS

// TODO: error handling
static void *osAlloc(void *hint UNUSED, size_t nbytes) {
#ifdef DDPOS_WINDOWS
	// TODO: use hint
	void *result = VirtualAlloc(NULL, nbytes, MEM_COMMIT, PAGE_READWRITE);
	if (result == NULL) {
		runtime_error_getlasterror(1, "VirtualAlloc fehlgeschlagen: %s");
	}
#else
	// TODO: use hint
	void *result = mmap(NULL, nbytes, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS, 0, 0);
	if (result == MAP_FAILED) {
		ddp_runtime_error(1, "mmap fehlgeschlagen: %s", strerror(errno));
	}
#endif
	DDP_DBGLOG("Allocated %p from OS", result);
	return result;
}

// TODO: error handling
static void osFree(void *p, UNUSED size_t nbytes) {
#ifdef DDPOS_WINDOWS
	if (VirtualFree(p, 0, MEM_RELEASE) == 0) {
		runtime_error_getlasterror(1, "VirtualFree fehlgeschlagen: %s");
	}
#else
	if (munmap(p, nbytes) < 0) {
		ddp_runtime_error(1, "munmap fehlgeschlagen: %s", strerror(errno));
	}
#endif
}

static void *ddp_reallocate_no_gc(void *pointer, size_t oldSize, size_t newSize) {
	bool collecting = gc.collecting;
	gc.collecting = true;
	void *result = ddp_reallocate(pointer, oldSize, newSize);
	gc.collecting = collecting;
	return result;
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
static GCSpan *allocate_span(GCTypeMeta objInfo) {
	DDP_DBGLOG("Allocating new span (GCTypeMeta: vtable %p, arrlen %d)", objInfo.vtable, objInfo.arrlen);
	DDP_DBGLOG("current spans (head: %p, tail: %p):", gc.spanHead, gc.spanTail);
	for (GCSpan *span = gc.spanHead; span != NULL; span = span->next) {
		DDP_DBGLOG("%p, %p -> %p", span->data, span, span->next);
	}

	GCSpan *newSpan = DDP_ALLOCATE_NO_GC(GCSpan, 1);
	newSpan->objInfo = objInfo;
	newSpan->objSize = objInfo.vtable->type_size * objInfo.arrlen;
	newSpan->allocatedSize = calculate_span_size(newSpan->objSize);
	newSpan->data = osAlloc(gc.spanTail == NULL ? NULL : &((uint8_t *)gc.spanTail->data)[newSpan->allocatedSize], newSpan->allocatedSize);
	newSpan->numObjs = newSpan->allocatedSize / newSpan->objSize;

	const size_t freeBytes = NUM_FREE_BYTES(newSpan->numObjs);
	const size_t markBytes = NUM_MARK_BYTES(newSpan->numObjs);
	newSpan->freeBits = DDP_ALLOCATE_NO_GC(uint8_t, freeBytes);
	memset(newSpan->freeBits, 0, freeBytes);
	newSpan->markBits = DDP_ALLOCATE_NO_GC(uint8_t, markBytes);
	memset(newSpan->markBits, 0, markBytes);
	newSpan->next = NULL;

	if (gc.spanHead == NULL) {
		gc.spanTail = gc.spanHead = newSpan;
	} else {
		gc.spanTail->next = newSpan;
		gc.spanTail = newSpan;
	}

	DDP_DBGLOG("Allocated new span");
	DDP_DBGLOG("new spans (head: %p, tail: %p):", gc.spanHead, gc.spanTail);
	for (GCSpan *span = gc.spanHead; span != NULL; span = span->next) {
		DDP_DBGLOG("%p, %p -> %p", span->data, span, span->next);
	}

	return newSpan;
}

static void free_span(GCSpan *span, GCSpan *prev) {
	DDP_DBGLOG("freeing span %p", span);

	GCSpan *next = span->next;
	if (prev) {
		prev->next = next;
	}
	if (gc.spanHead == span) {
		gc.spanHead = next;
	}
	if (gc.spanTail == span) {
		gc.spanTail = prev;
	}

	const size_t freeBytes = NUM_FREE_BYTES(span->numObjs);
	const size_t markBytes = NUM_MARK_BYTES(span->numObjs);
	DDP_FREE_ARRAY_NO_GC(uint8_t, span->freeBits, freeBytes);
	DDP_FREE_ARRAY_NO_GC(uint8_t, span->markBits, markBytes);
	osFree(span->data, span->allocatedSize);
	DDP_FREE_NO_GC(GCSpan, span);
}

static void clear_free_spans(void) {
	DDP_DBGLOG("clearing free spans");
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
static GCSpan *get_span_for_pointer(void *p) {
	for (GCSpan *span = gc.spanHead; span != NULL; span = span->next) {
		// maybe change this. see https://devblogs.microsoft.com/oldnewthing/20170927-00/?p=97095
		if ((uint8_t *)p >= (uint8_t *)span->data && (uint8_t *)p < ((uint8_t *)span->data + span->allocatedSize)) {
			return span;
		}
	}
	return NULL;
}

// returns the base object for a ref that is possibly in an array
static void *get_object_base_in_span(GCSpan *span, void *ref) {
	if (span->objInfo.arrlen == 1) {
		return ref;
	}

	ptrdiff_t base = (ptrdiff_t)span->data;
	ptrdiff_t step = span->objSize;
	ptrdiff_t val = (ptrdiff_t)ref;

	return (void *)(val - (((ptrdiff_t)ref - base) % step));
}

static void mark_node(void *ref, GCSpan *span, Color color) {
	DDP_DBGLOG("marking node %p with color %d", ref, color);

	unsigned index = (((((uint8_t *)ref) - ((uint8_t *)span->data)) / span->objSize));
	// always clear the bits
	bitmap_clear_two_bits(span->markBits, index);
	if (color != 0) {
		bitmap_set_two_bits(span->markBits, index, color);
	}
}

static Color get_color(void *ref, GCSpan *span) {
	unsigned index = (((((uint8_t *)ref) - ((uint8_t *)span->data)) / span->objSize));
	return bitmap_get_two_bits(span->markBits, index);
}

// TODO: use a binary search lookup table
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
		gc.global_roots = DDP_GROW_ARRAY_NO_GC(void **, gc.global_roots, gc.len_global_roots, gc.cap_global_roots);
	}

	gc.global_roots[gc.len_global_roots++] = root;
}

void ddp_register_gc_any_root(ddpany *root) {
	DDP_DBGLOG("Registering any root %p", root);

	if (gc.len_global_roots == gc.cap_global_roots) {
		gc.cap_global_roots += 8;
		gc.global_roots = DDP_GROW_ARRAY_NO_GC(void **, gc.global_roots, gc.len_global_roots, gc.cap_global_roots);
	}

	void **tagged_pointer = with_tag((void *)root, 1);

	gc.global_roots[gc.len_global_roots++] = tagged_pointer;
}

// returns zeroed memory
void *ddp_allocate_gc_ref(ddpvtable *vtable, ddpint arrlen) {
	DDP_DBGLOG("Allocating GC ref from vtable: %p, arrlen: %d", vtable, arrlen);

	if (arrlen <= 0 || vtable->type_size <= 0) {
		DDP_DBGLOG("returning NULL for arrlen == 0 or type_size == 0 ref allocation");
		return NULL;
	}

	GCTypeMeta objInfo = {.vtable = vtable, .arrlen = arrlen};
	void *space_slot = NULL;
	GCSpan *span = find_or_allocate_span_for_object(objInfo, &space_slot);

	unsigned index = (((uint8_t *)space_slot) - ((uint8_t *)span->data)) / span->objSize;
	bitmap_set_bit(span->freeBits, index);

	// zero the memory so that newly allocated spaces don't crash when being traced
	memset(space_slot, 0, span->objSize);

	DDP_DBGLOG("allocated ref: %p", space_slot);
	return space_slot;
}

// helper function which uses ddp_allocate_gc_ref to function similar to
// ddp_reallocate
void *ddp_reallocate_gc_ref(void *ptr, ddpvtable *vtable, ddpint oldArrLen,
							ddpint newArrLen) {
	void *newRef = ddp_allocate_gc_ref(vtable, newArrLen);

	if (newRef == NULL) {
		return newRef;
	}

	if (is_primitive_vtable(vtable)) {
		memcpy(newRef, ptr, vtable->type_size * oldArrLen);
	} else {
		for (ddpint i = 0; i < oldArrLen; i++) {
			vtable->deep_copy_func(&((uint8_t *)newRef)[i * vtable->type_size], &((uint8_t *)ptr)[i * vtable->type_size]);
		}
	}

	return newRef;
}

// pointer to the .llvm_stackmaps section
// defined by the compiler in the main object file, because the section itself is a local symbol
extern const uint8_t *__LLVM_StackMaps_External;

void ddp_init_gc(void) {
	DDP_DBGLOG("initializing gc");

	read_stack_map(&gc.stackMap, __LLVM_StackMaps_External);
	// dump_stackmap(&gc.stackMap);
	gc.global_roots = NULL;
	gc.len_global_roots = 0;
	gc.cap_global_roots = 0;

	gc.spanHead = NULL;
	gc.spanTail = NULL;

	gc.PAGE_SIZE = get_page_size();

	DDP_DBGLOG("done initializing gc");
}

static bool is_any_vtable(ddpvtable *vtable) {
	return vtable != NULL && vtable->free_func == (free_func_ptr)ddp_free_any;
}

static void trace_root(void *ref);
static void trace_any(ddpany *any) {
	DDP_DBGLOG("tracing any root %p", any);
	// don't trace NULL or Standardwert any
	if (any == NULL || any->vtable_ptr == NULL) {
		return;
	}

	DDP_DBGLOG("any->vtable_ptr: %p", any->vtable_ptr);
	DDP_DBGLOG("any->vtable_ptr->ptrmask: %p", any->vtable_ptr->ptrmask);
	DDP_DBGLOG("any->vtable_ptr->type_size: %p", any->vtable_ptr->type_size);

	const uint8_t *ptrmask = any->vtable_ptr->ptrmask;
	const void *ref = DDP_ANY_VALUE_PTR(any);

	DDP_DBGLOG("ptrmask: %hhu", ptrmask[0]);

	for (unsigned byte_index = 0; byte_index < PTRMASK_BYTES; byte_index++) {
		// byte-wise loop
		uint8_t byte = ptrmask[byte_index];
		// all 8 objects in this byte are free, so continue
		if (byte == 0) {
			continue;
		}

		for (int oneIndex = 0; oneIndex < 8; oneIndex++, byte = byte >> 1) {
			// remaining bits in this byte are 0
			if (byte == 0) {
				break;
			}

			// this quad is not a pointer
			if ((byte & 0x01) == 0) {
				continue;
			}

			unsigned index = oneIndex + byte_index * 8;
			void *nested_ref = ((void **)ref)[index];
			DDP_DBGLOG("tracing nested root %p", nested_ref);

			trace_root(nested_ref);
		}
	}
}

// TODO: arrays of references
static void trace_root(void *ref) {
	if (ref == NULL) {
		return;
	}
	DDP_DBGLOG("tracing root %p", ref);

	if (get_tag(ref) == 1) {
		DDP_DBGLOG("root is tagged, tracing any %p", ref);
		trace_any((ddpany *)without_tag(ref));
		return;
	}

	GCSpan *span = get_span_for_pointer(ref);
	if (span == NULL) {
		DDP_DBGLOG("No span found, not tracing");
		return;
	}

	ref = get_object_base_in_span(span, ref);
	DDP_DBGLOG("base pointer for ref: %p", ref);

	if (get_color(ref, span) == BLACK) {
		DDP_DBGLOG("not tracing already Black object");
		return;
	}

	mark_node(ref, span, GREY);

	if (is_any_vtable(span->objInfo.vtable)) {
		DDP_DBGLOG("tracing ddpany");
		// TODO: take arrlen and list capacity into account like in free_unmarked_objects
		for (ddpany *object = ref; object < (ddpany *)&((uint8_t *)ref)[span->objSize]; object = (ddpany *)&((uint8_t *)object)[span->objInfo.vtable->type_size]) {
			trace_any(object);
		}
	} else {
		const uint8_t *ptrmask = span->objInfo.vtable->ptrmask;

		for (unsigned byte_index = 0; byte_index < PTRMASK_BYTES; byte_index++) {
			// byte-wise loop
			uint8_t byte = ptrmask[byte_index];
			// all 8 objects in this byte are free, so continue
			if (byte == 0) {
				continue;
			}

			for (int oneIndex = 0; oneIndex < 8; oneIndex++, byte = byte >> 1) {
				// remaining bits in this byte are 0
				if (byte == 0) {
					break;
				}

				// this quad is not a pointer
				if ((byte & 0x01) == 0) {
					continue;
				}

				unsigned index = oneIndex + byte_index * 8;
				void *nested_ref = ((void **)ref)[index];
				DDP_DBGLOG("tracing nested root %p", nested_ref);
				// TODO: take arrlen and list capacity into account like in free_unmarked_objects
				// This for loop trace the root at this byte in the ptrmask for every object in its array if the span consists of arrays
				for (void *object = nested_ref; object < (void *)&((uint8_t *)nested_ref)[span->objSize]; object = &((uint8_t *)object)[span->objInfo.vtable->type_size]) {
					trace_root(object);
				}
			}
		}
	}

	mark_node(ref, span, BLACK);
}

static int get_unw_reg_number(uint16_t dwarf_reg) {
	switch (dwarf_reg) {
	case 0:
		return UNW_X86_64_RAX;
	case 1:
		return UNW_X86_64_RDX;
	case 2:
		return UNW_X86_64_RCX;
	case 3:
		return UNW_X86_64_RBX;
	case 4:
		return UNW_X86_64_RSI;
	case 5:
		return UNW_X86_64_RDI;
	case 6:
		return UNW_X86_64_RBP;
	case 7:
		return UNW_X86_64_RSP;
	case 8:
		return UNW_X86_64_R8;
	case 9:
		return UNW_X86_64_R9;
	case 10:
		return UNW_X86_64_R10;
	case 11:
		return UNW_X86_64_R11;
	case 12:
		return UNW_X86_64_R12;
	case 13:
		return UNW_X86_64_R13;
	case 14:
		return UNW_X86_64_R14;
	case 15:
		return UNW_X86_64_R15;
	case 17:
		return UNW_X86_64_XMM0;
	case 18:
		return UNW_X86_64_XMM1;
	case 19:
		return UNW_X86_64_XMM2;
	case 20:
		return UNW_X86_64_XMM3;
	case 21:
		return UNW_X86_64_XMM4;
	case 22:
		return UNW_X86_64_XMM5;
	case 23:
		return UNW_X86_64_XMM6;
	case 24:
		return UNW_X86_64_XMM7;
	case 25:
		return UNW_X86_64_XMM8;
	case 26:
		return UNW_X86_64_XMM9;
	case 27:
		return UNW_X86_64_XMM10;
	case 28:
		return UNW_X86_64_XMM11;
	case 29:
		return UNW_X86_64_XMM12;
	case 30:
		return UNW_X86_64_XMM13;
	case 31:
		return UNW_X86_64_XMM14;
	case 32:
		return UNW_X86_64_XMM15;
		// etc
	}
	return -1;
}

static inline ALWAYS_INLINE void mark_stack_roots(void) {
	DDP_DBGLOG("GC marking stack");
	unw_cursor_t cursor;
	unw_context_t context;

	unw_getcontext(&context);
	unw_init_local(&cursor, &context);

	while (unw_step(&cursor) > 0) {
		DDP_DBGLOG("Called unw_step");

		unw_word_t pc;
		if (unw_get_reg(&cursor, UNW_REG_IP, &pc) != 0) {
			DDP_DBGLOG("Could not read UNW_REG_IP");
			continue;
		}

		RecordAccessor record = find_stackmap_record(&gc.stackMap, pc);
		if (record == NULL) {
			continue;
		}

		unsigned numLocations = record_num_locations(record);
		for (unsigned i = 0; i < numLocations; i++) {
			void *root = NULL;
			LocationAccessor location = get_location(record, i);

			switch (location_kind(location)) {
			case LOC_REGISTER: {
				int regnum = get_unw_reg_number(location_dwarf_regnum(location));
				unw_word_t reg;
				if (unw_get_reg(&cursor, regnum, &reg) != 0) {
					DDP_DBGLOG("Could not read register %d", regnum);
				} else {
					root = (void *)reg;
				}
				DDP_DBGLOG("found register root %p", root);
				break;
			}
				// TODO: treat LOC_DIRECT and LOC_INDIRECT the same (same case)
			case LOC_DIRECT: {
				int regnum = get_unw_reg_number(location_dwarf_regnum(location));
				unw_word_t reg;
				if (unw_get_reg(&cursor, regnum, &reg) != 0) {
					DDP_DBGLOG("Could not read register %d", regnum);
					break;
				}

				int32_t offset = location_offset(location);

				void **addr = (void **)((uint8_t *)reg + offset);
				root = *addr;
				DDP_DBGLOG("found direct root %p %x %p", (void *)reg, offset, root);
				break;
			}
			case LOC_INDIRECT: {
				int regnum = get_unw_reg_number(location_dwarf_regnum(location));
				unw_word_t bp;
				if (unw_get_reg(&cursor, regnum, &bp) != 0) {
					DDP_DBGLOG("Could not read register %d", regnum);
					break;
				}

				int32_t offset = location_offset(location);

				void ***addr = (void ***)((uint8_t *)bp + offset);
				DDP_DBGLOG("indirect *addr: %p", *addr);
				root = *addr;

				DDP_DBGLOG("found indirect root %p %x %p", (void *)bp, offset, root);
				break;
			} break;
			case LOC_CONSTANT:
				root = (void *)(int64_t)location_offset(location);
				// DDP_DBGLOG("found constant root %p", root);
				break;
			case LOC_CONST_INDEX: {
				const Constant *constant = get_constant(&gc.stackMap, location_offset(location));
				root = (void *)constant->largeConstant;
				DDP_DBGLOG("found constant index root %p", root);
				break;
			}
			default: {
				DDP_DBGLOG("unknown location kind");
				break;
			}
			}

			trace_root(root);
		}
	}
}

static void mark_global_roots(void) {
	DDP_DBGLOG("GC marking global");
	for (void ***root = gc.global_roots; root != &gc.global_roots[gc.len_global_roots]; root++) {
		if (get_tag(*root) == 1) {
			ddpany *any_root = (ddpany *)(without_tag(*root));
			trace_any(any_root);
			continue;
		}

		if (*root != NULL) {
			DDP_DBGLOG("Root %p in use (Ref value: %p) (Span: %p)", *root, **root, get_span_for_pointer(**root));
		} else {
			DDP_DBGLOG("Root %p not in use", *root);
		}

		// ignore null refs
		if (*root == NULL) {
			continue;
		}

		trace_root(**root);
	}
}

static void free_unmarked_objects(void) {
	DDP_DBGLOG("GC sweeping objects");
	for (GCSpan *span = gc.spanHead; span != NULL; span = span->next) {
		free_func_ptr free_func = span->objInfo.vtable->free_func;

		// loop over and free every single white object
		for (unsigned byte_index = 0; byte_index < NUM_FREE_BYTES(span->numObjs); byte_index++) {
			// byte-wise loop
			uint8_t freeByte = span->freeBits[byte_index];
			// all 8 objects in this byte are free, so continue
			if (freeByte == 0) {
				continue;
			}

			// bit-wise loop
			for (int oneIndex = 0; oneIndex < 8; oneIndex++, freeByte = freeByte >> 1) {
				// all objects that are left in this byte are free, so break
				if (freeByte == 0) {
					break;
				}

				// there are objects left that are not free, but this particular one is, so continue
				if ((freeByte & 0x01) == 0) {
					continue;
				}

				unsigned index = oneIndex + byte_index * 8;

				DDP_DBGLOG("color of index %u is %d", index, bitmap_get_two_bits(span->markBits, index));
				if (bitmap_get_two_bits(span->markBits, index) != WHITE) {
					continue;
				}

				DDP_DBGLOG("freeing gc ref %p", (void *)&((uint8_t *)span->data)[index * span->objSize]);
				void *objectBase = (void *)&((uint8_t *)span->data)[index * span->objSize];
				if (free_func != NULL) {
					// TODO: for lists with unused capacity, this calls free for the unused slots as well, which should not happen
					for (void *object = objectBase; object < (void *)&((uint8_t *)objectBase)[span->objSize]; object = &((uint8_t *)object)[span->objInfo.vtable->type_size]) {
						// free the object
						free_func(object);
					}
				}
#ifdef DDP_GC_STRESS
				if (free_func == NULL) {
					for (void *object = objectBase; object < (void *)&((uint8_t *)objectBase)[span->objSize]; object = &((uint8_t *)object)[span->objInfo.vtable->type_size]) {
						// free the object
						memset(object, 0, span->objInfo.vtable->type_size);
					}
				}
#endif // DDP_GC_STRESS

				bitmap_clear_bit(span->freeBits, index);
			}
		}

		// mark all objects white before the next collection
		memset(span->markBits, WHITE, NUM_MARK_BYTES(span->numObjs));
	}
}

void ddp_gc(void) {
	if (gc.collecting) {
		return;
	}
	gc.collecting = true;

	DDP_DBGLOG("GC start =================================");

	mark_stack_roots();

	mark_global_roots();

	free_unmarked_objects();

	// TODO: don't always free all spans, reeuse them instead and only free at a certain limit
	clear_free_spans();

	gc.collecting = false;
	DDP_DBGLOG("GC end ---------------------------------");
}

void ddp_free_gc(void) {
	DDP_FREE_ARRAY_NO_GC(void *, gc.global_roots, gc.cap_global_roots);
	gc.global_roots = NULL;
	gc.len_global_roots = 0;
	gc.cap_global_roots = 0;
	// gc one last time with cleaned roots
	DDP_DBGLOG("last gc");
	ddp_gc();
	gc.collecting = true;

	free_stack_map(&gc.stackMap);
}
