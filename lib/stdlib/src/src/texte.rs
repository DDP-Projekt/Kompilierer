use crate::ddptypes::{DDPByte, DDPList, DDPString};

#[unsafe(no_mangle)]
pub extern "C" fn Text_Zu_ByteListe(text: *mut DDPString) -> DDPList<DDPByte> {
	DDPList::from(unsafe { Box::from_raw(text).as_bytes() })
}

#[unsafe(no_mangle)]
pub extern "C" fn ByteListe_Zu_Text(liste: *mut DDPList<DDPByte>) -> DDPString {
    let l = unsafe { Box::from_raw(liste) };
	DDPString {
		str: l.arr as *const i8,
		cap: l.len
	}
}