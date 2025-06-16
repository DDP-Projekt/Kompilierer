use ddpruntime::ddptypes::{DDPChar, DDPInt, DDPString};
use ddpruntime::ddp_reallocate;

#[repr(C)]
pub struct TextBauer {
	puffer: DDPString,
	laenge: DDPInt
}

#[unsafe(no_mangle)]
pub extern "C" fn Erhoehe_Kapazitaet(bauer: &mut TextBauer, cap: DDPInt) {
	unsafe {
		bauer.puffer.str = ddp_reallocate(bauer.puffer.str as *mut u8, bauer.puffer.cap, cap as usize) as *const i8;
		std::ptr::write_bytes(bauer.puffer.str.add(bauer.puffer.cap) as *mut u8, 0, cap as usize - bauer.puffer.cap);
		bauer.puffer.cap = cap as usize;
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn Bauer_Ende_Zeiger(bauer: &TextBauer) -> DDPInt {
	unsafe { bauer.puffer.str.add(bauer.laenge as usize) as DDPInt }
}

#[unsafe(no_mangle)]
pub extern "C" fn TextBauer_Als_Text(ret: &mut DDPString, bauer: &TextBauer) {
	unsafe { 
		*ret = DDPString::from_raw_parts(bauer.puffer.str as *const u8, bauer.laenge as usize);
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn TextBauer_Buchstabe_Anfuegen_C(bauer: &mut TextBauer, c: DDPChar) {
	unimplemented!()
}