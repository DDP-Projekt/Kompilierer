use std::ffi::{self, CStr};

use crate::ddptypes::{DDPBool, DDPChar, DDPInt, DDPString};

#[repr(C)]
pub struct TextIterator {
	ptr: *const ffi::c_char,
	end_ptr: *const ffi::c_char,
	text: *const DDPString,
	index: DDPInt
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_von_Text(text: *const DDPString) -> TextIterator {
	unsafe {
		TextIterator {
			ptr: (*text).str,
			end_ptr: if (*text).cap <= 0 {
				(*text).str
			} else {
				(*text).str.add((*text).cap - 1) 
			},
			text: text,
			index: 1
		}

	}
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Zuende(it: *const TextIterator) -> DDPBool {
	unsafe { (*it).ptr >= (*it).end_ptr }
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Buchstabe(it: *const TextIterator) -> DDPChar {
	if TextIterator_Zuende(it) {
		return 0;
	} 

	unsafe {
		let slice = CStr::from_ptr((*it).ptr).to_bytes();
		let ch = std::str::from_utf8(slice).ok().and_then(|s| s.chars().next()).unwrap_or('\0');
		ch as u32
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Naechster(it: *const TextIterator) {
	unimplemented!()
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Verbleibend(it: *const TextIterator) -> DDPInt {
	if TextIterator_Zuende(it) {
		return 0;
	}

	unimplemented!()
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Rest(it: *const TextIterator) -> DDPString {
	if TextIterator_Zuende(it) {
		return DDPString::from("");
	}

	unsafe {
		DDPString::from(CStr::from_ptr((*it).ptr).to_bytes())
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Bisher(it: *const TextIterator) -> DDPString {
	unimplemented!()
}