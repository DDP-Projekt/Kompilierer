use ddpruntime::ddptypes::{DDPChar, DDPInt, DDPString};
use ddpruntime::memory::ddp_reallocate;
use ddpruntime::utf8::utf8_char_to_string;
use debug_print::debug_println;

#[repr(C)]
pub struct TextBauer {
    puffer: DDPString,
    laenge: DDPInt,
}

#[unsafe(no_mangle)]
pub extern "C" fn Erhoehe_Kapazitaet(bauer: &mut TextBauer, cap: DDPInt) {
    debug_println!("Erhoehe_Kapazitaet {cap}");
    unsafe {
        bauer.puffer.str =
            ddp_reallocate(bauer.puffer.str as *mut u8, bauer.puffer.cap, cap as usize);
        std::ptr::write_bytes(
            bauer.puffer.str.add(bauer.puffer.cap) as *mut u8,
            0,
            cap as usize - bauer.puffer.cap,
        );
        bauer.puffer.cap = cap as usize;
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn Bauer_Ende_Zeiger(bauer: &TextBauer) -> DDPInt {
    let res = unsafe { bauer.puffer.str.add(bauer.laenge as usize) as DDPInt };
    debug_println!(
        "Bauer_Ende_Zeiger {} -> {}",
        bauer.puffer.str as DDPInt,
        res
    );
    res
}

#[unsafe(no_mangle)]
pub extern "C" fn TextBauer_Als_Text(ret: *mut DDPString, bauer: &TextBauer) {
    debug_println!("Als TExt");
    unsafe {
        std::ptr::write(
            ret,
            DDPString::from_raw_parts(bauer.puffer.str as *const u8, bauer.laenge as usize),
        );
    }
    debug_println!("Als Text done");
}

#[unsafe(no_mangle)]
pub extern "C" fn TextBauer_Buchstabe_Anfuegen_C(bauer: &mut TextBauer, c: DDPChar) {
    debug_println!("Buchstabe Anfügen");
    utf8_char_to_string(
        unsafe { bauer.puffer.str.cast_mut().add(bauer.laenge as usize) },
        c,
    );
    debug_println!("Buchstabe Anfügen done");
}
