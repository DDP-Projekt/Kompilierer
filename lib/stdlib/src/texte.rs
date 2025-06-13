use std::ptr::{null_mut};

use crate::ddptypes::{DDPByte, DDPList, DDPString};

unsafe extern "C" {
    fn ddp_reallocate(ptr: *mut u8, old_size: usize, new_size: usize) -> *mut u8;
}

#[unsafe(no_mangle)]
pub extern "C" fn Text_Zu_ByteListe(text: *mut DDPString) -> DDPList<DDPByte> {
	unsafe {
		let cap = (*text).cap-1;
		let ptr = ddp_reallocate(null_mut(), 0, cap);
		((*text).str as *const u8).copy_to(ptr, cap);

		DDPList { 
			arr: ptr,
			len: cap,
			cap: cap
		}
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn ByteListe_Zu_Text(liste: *mut DDPList<DDPByte>) -> DDPString {
	unsafe {
		let ptr = ddp_reallocate(null_mut(), 0, (*liste).len);
		(*liste).arr.copy_to(ptr, (*liste).len);

		DDPString {
			str: ptr as *const i8,
			cap: (*liste).len
		}
	}
}