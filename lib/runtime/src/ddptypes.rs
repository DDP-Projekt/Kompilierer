use debug_print::debug_println;

use crate::memory::{ddp_allocate, ddp_free, ddp_reallocate};
use crate::runtime::{ddp_panic, ddp_runtime_error};
use core::slice;
use std::ffi::{CStr, CString, c_char, c_void};
use std::ptr::{null, null_mut};
use std::{fmt, ptr, str};

pub type DDPInt = i64;
pub type DDPFloat = f64;
pub type DDPByte = u8;
pub type DDPChar = u32;
pub type DDPBool = bool;

#[derive(Debug)]
#[repr(C)]
pub struct DDPString {
    pub str: *const u8,
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
            let dst = ddp_allocate(len + 1);
            std::ptr::copy_nonoverlapping(ptr, dst, len);
            *dst.add(len) = 0; // add null terminator

            DDPString {
                str: dst,
                cap: len + 1,
            }
        }
    }

    // copies self into other and sets self to an empty string without dropping it
    // basically:
    //      *other = *self
    //      *self = DDP_EMPTY_STRING
    pub fn transfer_to(&mut self, other: *mut Self) {
        unsafe {
            ptr::copy_nonoverlapping(self as *mut DDPString, other, 1);
        }
        self.str = null();
        self.cap = 0;
    }

    pub fn to_string(self) -> String {
        unsafe { CString::from_raw(self.str as *mut i8) }
            .into_string()
            .unwrap()
    }

    pub fn as_slice(&self) -> Option<&[u8]> {
        if self.is_empty() {
            None
        } else {
            Some(unsafe { slice::from_raw_parts(self.str, self.cap) })
        }
    }

    pub fn as_cstr<'a>(&'a self) -> Option<&'a CStr> {
        if self.is_empty() {
            None
        } else {
            Some(unsafe { CStr::from_ptr(self.str as *const c_char) })
        }
    }

    pub fn to_str(&self) -> Result<Option<&str>, str::Utf8Error> {
        self.as_cstr()
            .map_or(Ok(None), |s| s.to_str().map(|s| Some(s)))
    }

    pub fn is_empty(&self) -> bool {
        self.str.is_null() || self.cap <= 0 || unsafe { self.str.read() == 0 }
    }

    pub fn byte_len(&self) -> DDPInt {
        if self.str.is_null() {
            0
        } else {
            self.cap as DDPInt
        }
    }

    pub fn len(&self) -> DDPInt {
        if self.is_empty() {
            0
        } else {
            self.to_str().unwrap().unwrap().chars().count() as DDPInt
        }
    }

    // helper function for certain inbuild functions
    // assumes 1-based index
    pub fn bounds_check(&self, index: usize) {
        if index as usize > self.cap || self.cap <= 1 {
            ddp_panic(
                1,
                format! {"Index außerhalb der Text Länge (Index war {index})\n"},
            );
        }
    }

    pub fn push(&mut self, c: char) {
        let mut tmp = [0u8; 4];
        let slice = c.encode_utf8(&mut tmp);

        if self.is_empty() {
            *self = DDPString::from(slice as &str);
            return;
        }

        self.str = ddp_reallocate(self.str.cast_mut(), self.cap, self.cap + slice.len());
        unsafe {
            ptr::copy_nonoverlapping(
                slice.as_ptr(),
                self.str.cast_mut().add(self.cap - 1),
                self.cap + slice.len(),
            );
            self.cap = self.cap + slice.len();
            ptr::write(self.str.cast_mut().add(self.cap - 1), 0);
        }
    }

    pub fn prepend(&mut self, c: char) {
        let mut tmp = [0u8; 4];
        let slice = c.encode_utf8(&mut tmp);

        if self.is_empty() {
            *self = DDPString::from(slice as &str);
            return;
        }

        self.str = ddp_reallocate(self.str.cast_mut(), self.cap, self.cap + slice.len());
        unsafe {
            ptr::copy(
                self.str.cast_mut(),
                self.str.cast_mut().add(slice.len()),
                self.cap,
            );
            ptr::copy_nonoverlapping(slice.as_ptr(), self.str.cast_mut(), slice.len());
            self.cap = self.cap + slice.len();
        }
    }
}

impl Drop for DDPString {
    fn drop(&mut self) {
        debug_println!("dropping string {:?}", self as *mut DDPString);
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
                str: ptr,
                cap: self.cap,
            }
        }
    }
}

impl PartialEq<Self> for DDPString {
    fn eq(&self, other: &Self) -> bool {
        ptr::eq(self, other)
            || self.byte_len() == other.byte_len()
                && (self.byte_len() == 0 || self.as_cstr().unwrap().eq(other.as_cstr().unwrap()))
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
        let str = CString::new(value).unwrap();
        Self {
            cap: str.count_bytes() + 1,
            str: str.into_raw() as *const u8,
        }
    }
}

impl fmt::Display for DDPString {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if self.str.is_null() {
            return write!(f, "");
        }
        match self.to_str() {
            Ok(s) => write!(f, "{}", s.unwrap_or("")),
            Err(e) => write!(f, "<{}>", e),
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_from_constant(ret: *mut DDPString, str: *const i8) {
    unsafe {
        ptr::write(
            ret,
            DDPString::from_raw_parts(str as *const u8, CStr::from_ptr(str).to_bytes().len()),
        );
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
    str1 == str2
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_length(str1: &DDPString) -> DDPInt {
    str1.len()
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_index(str: &DDPString, index: DDPInt) -> DDPChar {
    if index < 1 {
        ddp_panic(
            1,
            format!("Texte fangen bei Index 1 an. Es wurde wurde versucht {index} zu indizieren\n"),
        );
    }

    str.bounds_check(index as usize);

    match str
        .to_str()
        .unwrap()
        .unwrap()
        .chars()
        .nth(index as usize - 1)
    {
        Some(c) => c as DDPChar,
        None => ddp_panic(
            1,
            format!("Index außerhalb der Text Länge (Index war {index})\n"),
        ),
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_replace_char_in_string(str: &mut DDPString, ch: DDPChar, index: DDPInt) {
    if index < 1 {
        ddp_panic(
            1,
            format!("Texte fangen bei Index 1 an. Es wurde wurde versucht {index} zu indizieren\n"),
        );
    }
    str.bounds_check(index as usize);
    let index = index - 1;
    let mut tmp = [0u8; 4];
    let replacement = unsafe { char::from_u32_unchecked(ch) }.encode_utf8(&mut tmp);

    let str_slice = str.to_str().unwrap().unwrap();
    let start = str_slice.char_indices().nth(index as usize).unwrap();
    let end = start.0 + start.1.len_utf8();
    let start = start.0;

    // TODO: don't copy the whole string here
    let mut new = str.to_string();
    new.replace_range(start..end, &replacement);
    *str = DDPString::from(new);
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_slice(
    ret: *mut DDPString,
    str: &mut DDPString,
    index1: DDPInt,
    index2: DDPInt,
) {
    unsafe { ptr::write(ret, DDPString::new()) };

    if str.is_empty() {
        return;
    }

    let len = str.len();
    let index1 = index1.clamp(1, len);
    let index2 = index2.clamp(1, len);
    if index2 < index1 {
        ddp_panic(
            1,
            format!("Invalide Indexe (Index 1 war {index1}, Index 2 war {index2}\n"),
        );
    }

    let index1 = (index1 - 1) as usize;
    let index2 = (index2 - 1) as usize;

    let str_slice = str.to_str().unwrap().unwrap();
    let mut indices = str_slice.char_indices().map(|(i, _)| i);
    let start = indices.nth(index1).unwrap();
    let end = indices.nth(index2 - index1).unwrap_or(str_slice.len());
    unsafe { ptr::write(ret, DDPString::from(&str_slice[start..end])) };
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_string_verkettet(
    ret: *mut DDPString,
    str1: &mut DDPString,
    str2: &DDPString,
) {
    unsafe {
        ptr::write(ret, DDPString::new());
    }

    if str1.is_empty() && str2.is_empty() {
        return;
    } else if str1.is_empty() {
        unsafe {
            ptr::write(ret, str2.clone());
        }
        ddp_free_string(str1);
        return;
    } else if str2.is_empty() {
        str1.transfer_to(ret);
        return;
    }

    str1.str = ddp_reallocate(str1.str.cast_mut(), str1.cap, str1.cap - 1 + str2.cap);
    unsafe {
        ptr::copy_nonoverlapping(
            str2.str.cast_mut(),
            str1.str.cast_mut().add(str1.cap - 1),
            str2.cap,
        );
        str1.cap = str1.cap - 1 + str2.cap;
        str1.transfer_to(ret);
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_char_verkettet(ret: *mut DDPString, str: &mut DDPString, ch: DDPChar) {
    let ch = unsafe { char::from_u32_unchecked(ch) };
    str.push(ch);
    str.transfer_to(ret);
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_char_string_verkettet(ret: *mut DDPString, ch: DDPChar, str: &mut DDPString) {
    let ch = unsafe { char::from_u32_unchecked(ch) };
    str.prepend(ch);
    str.transfer_to(ret);
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_int_to_string(ret: *mut DDPString, i: DDPInt) {
    unsafe {
        ptr::write(ret, DDPString::from(i.to_string()));
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_float_to_string(ret: *mut DDPString, f: DDPFloat) {
    unsafe {
        ptr::write(ret, DDPString::from(format!("{f}").replacen(".", ",", 1)));
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_byte_to_string(ret: *mut DDPString, b: DDPByte) {
    unsafe {
        ptr::write(ret, DDPString::from(b.to_string()));
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_bool_to_string(ret: *mut DDPString, b: DDPBool) {
    unsafe {
        ptr::write(ret, DDPString::from(if b { "wahr" } else { "falsch" }));
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_char_to_string(ret: *mut DDPString, c: DDPChar) {
    unsafe {
        ptr::write(
            ret,
            DDPString::from(char::from_u32_unchecked(c).to_string()),
        );
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_to_int(str: &DDPString) -> DDPInt {
    if str.is_empty() {
        return 0;
    }

    str.to_str()
        .unwrap_or(None)
        .unwrap_or("")
        .parse::<DDPInt>()
        .unwrap_or(0)
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_string_to_float(str: &DDPString) -> DDPFloat {
    if str.is_empty() {
        return 0.0;
    }

    str.to_str()
        .unwrap_or(None)
        .unwrap_or("")
        .replacen(",", ".", 1)
        .parse::<DDPFloat>()
        .unwrap_or(0.0)
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
