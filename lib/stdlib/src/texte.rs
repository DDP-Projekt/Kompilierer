use ddpruntime::ddptypes::{DDPByte, DDPList, DDPString};

#[unsafe(no_mangle)]
pub extern "C" fn Text_Zu_ByteListe(ret: &mut DDPList<DDPByte>, text: &DDPString) {
	if text.is_empty() {
		return *ret = DDPList::new();
	}

	unsafe {
		*ret = DDPList::from_raw_parts(text.str as *const u8, text.cap-1)
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn ByteListe_Zu_Text(ret: &mut DDPString, liste: &DDPList<DDPByte>) {
	if liste.len == 0 {
		return *ret = DDPString::new();
	}

	unsafe {
		*ret = DDPString::from_raw_parts(liste.arr, liste.len as usize);
	}
}