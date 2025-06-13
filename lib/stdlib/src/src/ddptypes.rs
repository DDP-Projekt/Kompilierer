use std::ffi::CStr;
use std::fmt;
use std::ffi;

pub type DDPInt = i64;
pub type DDPFloat = f64;
pub type DDPByte = u8;
pub type DDPChar = u32;
pub type DDPBool = bool;
pub type DDPAny = ffi::c_void;
 
#[repr(C)]
pub struct DDPString {
	pub str: *const ffi::c_char,
	pub cap: usize
}

impl DDPString {
	pub unsafe fn as_bytes(&self) -> &'static [u8] {
		unsafe { std::slice::from_raw_parts(self.str as *const u8, self.cap) }
	}
}

impl From<&[u8]> for DDPString {
	fn from(value: &[u8]) -> Self {
		let cstr = CStr::from_bytes_with_nul(value).unwrap();
		DDPString {
			str: cstr.as_ptr(),
			cap: value.len()
		}
	}
}

impl From<&str> for DDPString {
	fn from(value: &str) -> Self {
		DDPString::from(value.as_bytes())
	}
}

impl From<String> for DDPString {
	fn from(value: String) -> Self {
		DDPString::from(value.as_bytes())
	}
}

impl fmt::Display for DDPString {
	fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
		if self.str.is_null() {
            return write!(f, "<null>");
        }
        unsafe {
            match ffi::CStr::from_ptr(self.str).to_str() {
                Ok(s) => write!(f, "{}", s),
                Err(e) => write!(f, "<{}>", e),
            }
        }
	}
}

#[repr(C)]
pub struct DDPList<T> {
	pub arr: *const T,
	pub len: usize,
	pub cap: usize
}

impl<T> From<&[T]> for DDPList<T> {
	fn from(value: &[T]) -> Self {
		DDPList { 
			arr: value.as_ptr(), 
			len: value.len(),
			cap: value.len()
		}
	}
}