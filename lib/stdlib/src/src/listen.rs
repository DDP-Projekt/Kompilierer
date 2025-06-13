use crate::ddptypes::{DDPAny, DDPChar, DDPList, DDPString};

#[unsafe(no_mangle)]
pub extern "C" fn Aneinandergehaengt_Buchstabe_Ref(liste: *mut DDPList<DDPChar>) -> DDPString {
	unimplemented!();
}

#[unsafe(no_mangle)]
pub extern "C" fn efficient_list_append(list: *mut DDPList<DDPAny>, elem: *const DDPAny, any: *const DDPAny) -> DDPString {
	unimplemented!();
}