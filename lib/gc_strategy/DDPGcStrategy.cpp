#include "llvm/IR/DerivedTypes.h"
#include "llvm/IR/GCStrategy.h"

using namespace llvm;

namespace {

class DDPGcStrategy : public GCStrategy {
public:
  DDPGcStrategy() {
    UseStatepoints = true;
    UseRS4GC = true;

    NeededSafePoints = false;
  }

  virtual std::optional<bool> isGCManagedPointer(const Type *Ty) const {
    const PointerType *PT = cast<PointerType>(Ty);
    return PT->getAddressSpace() == 1;
  }
};

} // namespace

extern "C" void ddp_initialize_gc_strategy() {
  static GCRegistry::Add<DDPGcStrategy> DDP_GC("ddp-gc", "The DDP GC Strategy");
}
