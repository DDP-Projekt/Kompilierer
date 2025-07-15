use core::slice;
use std::ffi::c_int;

#[unsafe(no_mangle)]
pub unsafe extern "C" fn utf8_indicated_num_bytes(c: u8) -> c_int {
    match c.trailing_zeros() {
        7 => 1,
        6 => 2,
        3 => 3,
        4 => 4,
        _ => 0,
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn utf8_string_to_char(str: *const u8, out: *mut u32) -> usize {
    if str.is_null() {
        return usize::MAX;
    }
    let num_bytes = unsafe { utf8_indicated_num_bytes(str.read()) };

    // Read up to 4 bytes from the pointer (max size of a UTF-8 codepoint)
    let bytes = unsafe { slice::from_raw_parts(str, num_bytes as usize) };

    // Try to decode the first character
    if let Ok(s) = str::from_utf8(bytes) {
        if let Some(c) = s.chars().next() {
            unsafe {
                out.write(c as u32);
            }
            return c.len_utf8();
        };
    }
    usize::MAX
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn utf8_char_to_string(str: *mut u8, c: u32) -> usize {
    let ch = match std::char::from_u32(c) {
        Some(ch) => ch,
        None => unsafe {
            str.write(0);
            return usize::MAX;
        },
    };

    let buf = &mut [0u8; 4];
    let encoded = ch.encode_utf8(buf);

    unsafe {
        std::ptr::copy_nonoverlapping(encoded.as_ptr(), str, encoded.len());
    }

    ch.len_utf8()
}
