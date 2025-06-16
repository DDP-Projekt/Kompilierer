use std::fmt;
use std::ffi;
use std::ffi::CStr;
use std::ptr::{null, null_mut};
use crate::ddp_reallocate;

pub type DDPInt = i64;
pub type DDPFloat = f64;
pub type DDPByte = u8;
pub type DDPChar = u32;
pub type DDPBool = bool;
pub type DDPAny = ffi::c_void;

#[derive(Debug)]
#[repr(C)]
pub struct DDPString {
	pub str: *const ffi::c_char,
	pub cap: usize
}

impl DDPString {
	// creates an empty String
	pub fn new() -> DDPString {
		DDPString { 
			str: null(),
			cap: 0
		}
	}

	/// allocates a new DDP String using the given buffer and length\
	/// WARNING: BUFFER SHOULD NOT BE NULL TERMINATED
	pub unsafe fn from_raw_parts(ptr: *const u8, len: usize) -> DDPString {
		if ptr.is_null() {
			panic!("ptr was null")
		}

		unsafe {
			let dst = ddp_reallocate(null_mut(), 0, len+1);
			std::ptr::copy_nonoverlapping(ptr, dst, len);
			*dst.add(len) = 0; // add null terminator

			DDPString {
				str: dst as *const i8,
				cap: len+1
			}
		}
	}

	pub fn is_empty(&self) -> bool {
		self.str.is_null() || self.cap <= 0 || unsafe { self.str.read() == 0 }
	}

	pub fn byte_len(&self) -> DDPInt {
		if self.str.is_null() {
			0
		}
		else {
			unsafe { CStr::from_ptr(self.str) }.to_bytes().len() as DDPInt
		}
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

			Self { str: ptr as *const i8, cap: self.cap }
		}
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

#[derive(Debug)]
#[repr(C)]
pub struct DDPList<T> {
	pub arr: *const T,
	pub len: i64,
	pub cap: i64
}

impl<T> DDPList<T> {
	pub fn new() -> DDPList<T> {
		DDPList { arr: null(), len: 0, cap: 0 }
	}

	pub unsafe fn from_raw_parts(ptr: *const T, len: usize) -> DDPList<T> {
		unsafe {
			let dst = ddp_reallocate(null_mut(), 0, len);
			std::ptr::copy_nonoverlapping(ptr, dst as *mut T, len);	
		
			DDPList {
				arr: dst as *const T,
				len: len as i64,
				cap: len as i64
			}
		}
	}
}

impl<T> From<&[T]> for DDPList<T> {
	fn from(value: &[T]) -> Self {
		unsafe { DDPList::from_raw_parts(value.as_ptr(), value.len()) }
	}
}