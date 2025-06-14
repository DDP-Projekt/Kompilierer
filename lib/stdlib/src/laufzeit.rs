use std::{io::IsTerminal, process::exit};

use crate::{ddp_end_runtime, ddp_runtime_error, ddptypes::*};

#[unsafe(no_mangle)]
pub extern "C" fn Programm_Beenden(code: DDPInt) {
	unsafe { ddp_end_runtime(); }
	exit(code as i32)
}

#[unsafe(no_mangle)]
pub extern "C" fn Laufzeitfehler(nachricht: &DDPString, code: DDPInt) {
	unsafe { ddp_runtime_error(code, nachricht.str); }
}

#[unsafe(no_mangle)]
pub extern "C" fn Ist_Befehlszeile() -> DDPBool {
	std::io::stdout().is_terminal()
}

#[unsafe(no_mangle)]
pub extern "C" fn Betriebssystem(ret: &mut DDPString) {
	*ret = DDPString::from(std::env::consts::OS)
}

#[unsafe(no_mangle)]
pub extern "C" fn Arbeitsverzeichnis(ret: &mut DDPString) {
	let cwd = std::env::current_dir().unwrap();
	let cwd_str = cwd.to_str().unwrap();
	*ret = DDPString::from(cwd_str)
}