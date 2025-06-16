use std::io::{self, Write};
use ddpruntime::ddptypes::*;

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Zahl(x: DDPInt) {
	print!("{x}");
	let _ = io::stdout().flush();
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Kommazahl(x: DDPFloat) {
	if x.is_infinite() {
		print!("{}Unendlich", if x.is_sign_positive() { "" } else { "-" })
	} else if x.is_nan() {
		print!("Keine Zahl (NaN)")
	} else {
		print!("{x}")
	}
	let _ = io::stdout().flush();
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Byte(x: DDPByte) {
	print!("{x}");
	let _ = io::stdout().flush();
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Wahrheitswert(x: DDPBool) {
	print!("{}", if x { "wahr" } else { "falsch" });
	let _ = io::stdout().flush();
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Buchstabe(x: DDPChar) {
	print!("{}", char::from_u32(x).unwrap());
	let _ = io::stdout().flush();
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Text(x: &DDPString) {
	print!("{}", x);
	let _ = io::stdout().flush();
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Fehler(x: &DDPString) {
	eprintln!("{}", x);
	let _ = io::stdout().flush();
}