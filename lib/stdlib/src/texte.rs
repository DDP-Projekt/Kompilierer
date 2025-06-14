use crate::ddptypes::{DDPByte, DDPList, DDPString};

#[unsafe(no_mangle)]
pub extern "C" fn Text_Zu_ByteListe(ret: *mut DDPList<DDPByte>, text: &DDPString) {
	unsafe {
		if text.is_empty() {
			return ret.write(DDPList::new());
		}

		ret.write(DDPList::from_raw_parts(text.str as *const u8, text.cap-1))
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn ByteListe_Zu_Text(ret: *mut DDPString, liste: *mut DDPList<DDPByte>) {
	unsafe {
		if (*liste).len == 0 {
			return ret.write(DDPString::new());
		}
		
		ret.write(DDPString::from_raw_parts((*liste).arr, (*liste).len as usize));
	}
}