use ddpruntime::ddptypes::{DDPChar, DDPInt, DDPString};
use ddpruntime::memory::ddp_reallocate;
use std::ptr::null_mut;

#[unsafe(no_mangle)]
pub extern "C" fn C_Memcpy(dest: DDPInt, src: DDPInt, size: DDPInt) {
    unsafe {
        std::ptr::copy_nonoverlapping(src as *mut u64, dest as *mut u64, size as usize);
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn Text_Zu_CString(t: &DDPString) -> DDPInt {
    t.str.addr() as DDPInt
}

#[unsafe(no_mangle)]
pub extern "C" fn Text_Zu_Zeiger(t: *const DDPString) -> DDPInt {
    t.addr() as DDPInt
}

#[unsafe(no_mangle)]
pub extern "C" fn Erstelle_Byte_Puffer(ret: &mut DDPString, n: DDPInt) {
    unsafe {
        let buf = ddp_reallocate(null_mut(), 0, (n + 1) as usize);
        (*buf.add(n as usize)) = 0;

        ret.cap = (n + 1) as usize;
        ret.str = buf;
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn Text_Byte_Groesse(t: &DDPString) -> DDPInt {
    if t.cap > 0 { t.cap as i64 - 1 } else { 0 }
}

#[unsafe(no_mangle)]
pub extern "C" fn Buchstabe_Byte_Groesse(c: DDPChar) -> DDPInt {
    core::char::from_u32(c).unwrap().len_utf8() as DDPInt
}
