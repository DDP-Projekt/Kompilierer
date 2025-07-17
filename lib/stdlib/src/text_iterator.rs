use ddpruntime::ddptypes::{DDPBool, DDPChar, DDPInt, DDPString};
use std::ffi::{CStr, c_char};

#[repr(C)]
pub struct TextIterator {
    ptr: *const c_char,
    end_ptr: *const c_char,
    text: *const DDPString,
    index: DDPInt,
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_von_Text(ret: &mut TextIterator, text: &DDPString) {
    *ret = TextIterator {
        ptr: text.str as *const c_char,
        end_ptr: if text.cap <= 0 {
            text.str as *const c_char
        } else {
            unsafe { text.str.add(text.cap - 1) as *const c_char }
        },
        text: text,
        index: 1,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Zuende(it: &TextIterator) -> DDPBool {
    it.ptr >= it.end_ptr
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Buchstabe(it: &TextIterator) -> DDPChar {
    if TextIterator_Zuende(it) {
        return 0;
    }

    let slice = unsafe { CStr::from_ptr((*it).ptr).to_bytes() };
    std::str::from_utf8(slice)
        .ok()
        .and_then(|s| s.chars().next())
        .unwrap_or('\0') as u32
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Naechster(it: &mut TextIterator) -> DDPChar {
    if TextIterator_Zuende(it) {
        return 0;
    }

    unsafe {
        match CStr::from_ptr(it.ptr).to_str() {
            Ok(s) => {
                if let Some(c) = s.chars().next() {
                    it.ptr = it.ptr.add(c.len_utf8());
                    it.index += 1;
                    c as u32
                } else {
                    panic!("Empty string")
                }
            }
            Err(e) => panic!("{e}"),
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Verbleibend(it: &TextIterator) -> DDPInt {
    if TextIterator_Zuende(it) {
        return 0;
    }

    unsafe { CStr::from_ptr(it.ptr).to_str().unwrap().len() as DDPInt }
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Rest(ret: &mut DDPString, it: &TextIterator) {
    if TextIterator_Zuende(it) {
        return *ret = DDPString::new();
    }

    *ret = DDPString::from(unsafe { CStr::from_ptr(it.ptr) }.to_bytes())
}

#[unsafe(no_mangle)]
pub extern "C" fn TextIterator_Bisher(ret: &mut DDPString, it: &TextIterator) {
    unsafe {
        if it.ptr <= (*it.text).str as *const c_char {
            return;
        }

        let cap = it.ptr.sub((*it.text).str.addr()).addr();
        *ret = DDPString::from_raw_parts((*it.text).str as *const u8, cap)
    }
}
