use core::slice;
use std::ffi::c_int;

#[unsafe(no_mangle)]
pub extern "C" fn utf8_indicated_num_bytes(c: u8) -> c_int {
    match c.leading_ones() {
        0 => 1,
        2 => 2,
        3 => 3,
        4 => 4,
        _ => 0,
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn utf8_string_to_char(str: *const u8, out: *mut u32) -> usize {
    if str.is_null() {
        return usize::MAX;
    }
    let num_bytes = utf8_indicated_num_bytes(unsafe { str.read() });

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

#[cfg(test)]
mod tests {
    use crate::utf8::utf8_string_to_char;

    #[test]
    fn test_utf8_string_to_char() {
        let mut s: [u8; 5] = [0, 0, 0, 0, 0];
        'Ü'.encode_utf8(s.as_mut_slice());

        let mut c: u32 = 0;
        let ps: *const u8 = s.as_ptr();
        let pc: *mut u32 = &mut c;
        assert_eq!(utf8_string_to_char(ps, pc), 2);
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn utf8_char_to_string(str: *mut u8, c: u32) -> usize {
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
