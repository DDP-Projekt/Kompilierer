use std::alloc::{GlobalAlloc, Layout, System};
use std::ffi::c_int;
use std::ptr::null_mut;
use std::sync::atomic::{AtomicUsize, Ordering::Relaxed};

struct DDPAlloc;

static ALLOCATED: AtomicUsize = AtomicUsize::new(0);

unsafe impl GlobalAlloc for DDPAlloc {
    unsafe fn alloc(&self, layout: Layout) -> *mut u8 {
        let ret = unsafe { System.alloc(layout) };
        if !ret.is_null() {
            ALLOCATED.fetch_add(layout.size(), Relaxed);
        }
        ret
    }

    unsafe fn dealloc(&self, ptr: *mut u8, layout: Layout) {
        unsafe {
            System.dealloc(ptr, layout);
        }
        ALLOCATED.fetch_sub(layout.size(), Relaxed);
    }
}

#[global_allocator]
static DDP_ALLOC: DDPAlloc = DDPAlloc;

unsafe extern "C" {
    fn ddp_runtime_error(code: c_int, fmt: *const u8, ...);
}

const DDP_DEFAULT_ALIGN: usize = 8;

fn check_null(ptr: *mut u8) -> *mut u8 {
    match ptr {
        result if result.is_null() => {
            unsafe { ddp_runtime_error(1, "out of memory\n".as_ptr()) };
            unreachable!("ddp_runtime_error");
        }
        result => result,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_reallocate(ptr: *mut u8, old_size: usize, new_size: usize) -> *mut u8 {
    match (ptr, old_size, new_size) {
        // new_size == 0 means free
        (_, _, 0) => {
            unsafe {
                DDP_ALLOC.dealloc(
                    ptr,
                    Layout::from_size_align(old_size, DDP_DEFAULT_ALIGN).unwrap(),
                )
            };
            return null_mut();
        }
        (ptr, old, new) if old == new => ptr,
        (ptr, _, _) if ptr.is_null() => unsafe {
            check_null(
                DDP_ALLOC.alloc(Layout::from_size_align(new_size, DDP_DEFAULT_ALIGN).unwrap()),
            )
        },
        (ptr, old_size, new_size) => unsafe {
            check_null(DDP_ALLOC.realloc(
                ptr,
                Layout::from_size_align(old_size, DDP_DEFAULT_ALIGN).unwrap(),
                new_size,
            ))
        },
    }
}
