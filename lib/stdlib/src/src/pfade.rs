use std::{fs::canonicalize, path};

use crate::ddptypes::DDPString;

#[unsafe(no_mangle)]
pub extern "C" fn Windows_Saeubern(pfad: DDPString) -> DDPString {
	unimplemented!()
}

#[unsafe(no_mangle)]
pub extern "C" fn Windows_Pfad_Verbinden(p1: DDPString, p2: DDPString) -> DDPString {
    unimplemented!()
}