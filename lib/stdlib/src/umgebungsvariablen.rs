use std::env;

use crate::ddptypes::DDPString;

#[unsafe(no_mangle)]
pub extern "C" fn Hole_Umgebungsvariable(ret: *mut DDPString, name: &DDPString) {
    unsafe {
		ret.write(DDPString::from(env::var(name.to_string()).unwrap()))
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn Setze_Umgebungsvariable(name: &DDPString, wert: &DDPString) {
	unsafe {
		env::set_var(name.to_string(), wert.to_string())
	}
}
