use debug_print::debug_println;

use crate::memory::{ddp_allocate, ddp_free, ddp_reallocate};
use crate::runtime::ddp_runtime_error;
use core::slice;
use std::ffi::{CStr, c_void};
use std::ptr::{null, null_mut};
use std::{fmt, ptr};

pub type DDPInt = i64;
pub type DDPFloat = f64;
pub type DDPByte = u8;
pub type DDPChar = u32;
pub type DDPBool = bool;

#[derive(Debug)]
#[repr(C)]
pub struct DDPString {
    pub str: *const i8,
    pub cap: usize,
}

impl DDPString {
    // creates an empty String
    pub fn new() -> DDPString {
        DDPString {
            str: null(),
            cap: 0,
        }
    }

    /// allocates a new DDP String using the given buffer and length\
    /// WARNING: BUFFER SHOULD NOT BE NULL TERMINATED
    pub unsafe fn from_raw_parts(ptr: *const u8, len: usize) -> DDPString {
        if ptr.is_null() {
            unsafe {
                ddp_runtime_error(1, "ptr was null".as_ptr());
            }
        }

        unsafe {
            debug_println!("allocating string");
            let dst = ddp_allocate(len + 1);
            std::ptr::copy_nonoverlapping(ptr, dst, len);
            *dst.add(len) = 0; // add null terminator

            debug_println!("done allocating string");
            DDPString {
                str: dst as *const i8,
                cap: len + 1,
            }
        }
    }

    pub fn is_empty(&self) -> bool {
        self.str.is_null() || self.cap <= 0 || unsafe { self.str.read() == 0 }
    }

    pub fn byte_len(&self) -> DDPInt {
        if self.str.is_null() {
            0
        } else {
            unsafe { CStr::from_ptr(self.str) }.to_bytes().len() as DDPInt
        }
    }
}

impl Drop for DDPString {
    fn drop(&mut self) {
        debug_println!("\tdroping string {self:?}\n");
        ddp_free_string(self);
    }
}

impl Clone for DDPString {
    fn clone(&self) -> Self {
        if self.str.is_null() {
            return DDPString::new();
        }

        unsafe {
            let ptr = ddp_reallocate(null_mut(), 0, self.cap);
            std::ptr::copy_nonoverlapping(self.str as *const u8, ptr, self.cap);

            Self {
                str: ptr as *const i8,
                cap: self.cap,
            }
        }
    }
}

impl PartialEq<Self> for DDPString {
    fn eq(&self, other: &Self) -> bool {
        debug_println!("comparing strings");
        ptr::eq(self, other)
            || !(self.byte_len() != other.byte_len())
                && unsafe { CStr::from_ptr(self.str).eq(CStr::from_ptr(other.str)) }
    }
}

impl From<&[u8]> for DDPString {
    /// allocates a new DDP String from a u8 slice
    fn from(value: &[u8]) -> Self {
        unsafe { DDPString::from_raw_parts(value.as_ptr(), value.len()) }
    }
}

impl From<&str> for DDPString {
    /// allocates a new DDP String from a str
    fn from(value: &str) -> Self {
        unsafe { DDPString::from_raw_parts(value.as_ptr(), value.len()) }
    }
}

impl From<String> for DDPString {
    /// allocates a new DDP String from a String
    fn from(value: String) -> Self {
        unsafe { DDPString::from_raw_parts(value.as_ptr(), value.len()) }
    }
}

impl fmt::Display for DDPString {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if self.str.is_null() {
            return write!(f, "");
        }
        unsafe {
            match CStr::from_ptr(self.str).to_str() {
                Ok(s) => write!(f, "{}", s),
                Err(e) => write!(f, "<{}>", e),
            }
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_from_constant(ret: *mut DDPString, str: *const i8) {
    unsafe {
        debug_println!("\tstring constant start");
        ptr::write(ret, DDPString::from_raw_parts(str as *const u8, CStr::from_ptr(str).to_bytes().len()));
        debug_println!("\tstring constant done");
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_free_string(str: &mut DDPString) {
    ddp_free(str.str as *mut u8, str.cap);
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_deep_copy_string(ret: *mut DDPString, str: &mut DDPString) {
    unsafe { 
        ptr::write(ret, str.clone());
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_empy(str: &mut DDPString) -> DDPBool {
    str.is_empty()
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_strlen(str: &mut DDPString) -> DDPInt {
    str.byte_len()
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_equal(str1: &mut DDPString, str2: &mut DDPString) -> DDPBool {
    debug_println!("comparing strings {str1:?} {str2:?}");
    str1 == str2
}

#[derive(Debug)]
#[repr(C)]
pub struct DDPList<T> {
    pub arr: *const T,
    pub len: i64,
    pub cap: i64,
}

impl<T> DDPList<T> {
    pub fn new() -> DDPList<T> {
        DDPList {
            arr: null(),
            len: 0,
            cap: 0,
        }
    }

    pub unsafe fn from_raw_parts(ptr: *const T, len: usize) -> DDPList<T> {
        unsafe {
            let dst = ddp_reallocate(null_mut(), 0, len);
            std::ptr::copy_nonoverlapping(ptr, dst as *mut T, len);

            DDPList {
                arr: dst as *const T,
                len: len as i64,
                cap: len as i64,
            }
        }
    }
}

impl<T> From<&[T]> for DDPList<T> {
    fn from(value: &[T]) -> Self {
        unsafe { DDPList::from_raw_parts(value.as_ptr(), value.len()) }
    }
}

type DDPFreeFunc = Option<unsafe extern "C" fn(val: *mut c_void)>;
type DDPDeepCopyFunc = Option<unsafe extern "C" fn(ret: *mut c_void, val: *mut c_void)>;
type DDPEqualFunc = Option<unsafe extern "C" fn(val1: *mut c_void, val2: *mut c_void) -> bool>;

#[derive(Debug)]
#[repr(C)]
pub struct DDPVTable {
    pub type_size: DDPInt,
    pub free_func: DDPFreeFunc,
    pub deep_copy_func: DDPDeepCopyFunc,
    pub equal_func: DDPEqualFunc,
}

impl DDPVTable {
    pub fn is_primitive(&self) -> bool {
        self.free_func.is_none()
    }
}

#[repr(C)]
pub union DDPAnyValue {
    pub value_ptr: *mut c_void,
    pub value: [u8; 16],
}

#[repr(C)]
pub struct DDPAny {
    pub vtable_ptr: *const DDPVTable,
    pub value: DDPAnyValue,
}

impl DDPAny {
    pub fn new() -> Self {
        DDPAny {
            vtable_ptr: null(),
            value: DDPAnyValue {
                value_ptr: null_mut(),
            },
        }
    }

    pub fn is_small(&self) -> bool {
        unsafe { (*self.vtable_ptr).type_size <= 16 }
    }

    pub fn value_ptr(&self) -> *mut c_void {
        if self.is_small() {
            unsafe { self.value.value.as_ptr() as *mut c_void }
        } else {
            unsafe { self.value.value_ptr }
        }
    }

    pub fn is_standard_value(&self) -> bool {
        self.vtable_ptr.is_null()
    }

    pub fn is_primitive(&self) -> bool {
        !self.is_standard_value() && unsafe { self.vtable_ptr.read().is_primitive() }
    }
}

impl Drop for DDPAny {
    fn drop(&mut self) {
        ddp_free_any(self)
    }
}

impl Clone for DDPAny {
    fn clone(&self) -> Self {
        let mut result = DDPAny::new();
        ddp_deep_copy_any(&mut result, &self);
        result
    }
}

impl PartialEq<Self> for DDPAny {
    fn eq(&self, other: &Self) -> bool {
        ddp_any_equal(&self, &other)
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_free_any(any: &mut DDPAny) {
    if any.is_standard_value() {
        return;
    }

    // free the underlying value
    if !any.is_primitive() {
        unsafe { (any.vtable_ptr.read().free_func.unwrap())(any.value_ptr()) }
    }

    // free the memory allocated for the value itself
    if !any.is_small() && !any.is_standard_value() {
        ddp_free(any.value_ptr() as *mut u8, unsafe {
            any.vtable_ptr.read().type_size as usize
        });
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_deep_copy_any(ret: &mut DDPAny, any: &DDPAny) {
    ret.vtable_ptr = any.vtable_ptr;

    if ret.is_standard_value() {
        return;
    }

    if !ret.is_small() {
        ret.value.value_ptr =
            ddp_allocate(unsafe { ret.vtable_ptr.read().type_size as usize }) as *mut c_void;
    }

    if ret.is_primitive() {
        unsafe {
            std::ptr::copy_nonoverlapping(
                any.value_ptr(),
                ret.value_ptr(),
                ret.vtable_ptr.read().type_size as usize,
            );
        }
    } else {
        unsafe { (ret.vtable_ptr.read().deep_copy_func.unwrap())(ret.value_ptr(), any.value_ptr()) }
    }
}
#[unsafe(no_mangle)]
pub extern "C" fn ddp_any_equal(any1: &DDPAny, any2: &DDPAny) -> bool {
    if !ptr::eq(any1.vtable_ptr, any2.vtable_ptr) {
        return false;
    }

    if any1.is_standard_value() {
        return true;
    }

    let any2_size = if any2.is_standard_value() {
        0
    } else {
        unsafe { any2.vtable_ptr.read().type_size }
    };

    if unsafe { any1.vtable_ptr.read().type_size } != any2_size {
        return false;
    }

    if any1.is_primitive() {
        return unsafe {
            slice::from_raw_parts(any1.value_ptr() as *const u8, any2_size as usize)
                == slice::from_raw_parts(any2.value_ptr() as *const u8, any2_size as usize)
        };
    }

    unsafe { (any1.vtable_ptr.read().equal_func.unwrap())(any1.value_ptr(), any2.value_ptr()) }
}
