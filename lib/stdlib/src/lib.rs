use crate::ddptypes::DDPInt;

mod ddptypes;
mod ausgabe;
mod mathe;
mod zufall;
mod zeit;
mod kryptographie;
mod umgebungsvariablen;
mod pfade;
mod texte;
mod text_iterator;
mod listen;
mod laufzeit;
mod text_bauer;
mod c;

unsafe extern "C" {
	pub unsafe fn ddp_reallocate(ptr: *mut u8, old_size: usize, new_size: usize) -> *mut u8;
	pub unsafe fn ddp_end_runtime();
	pub unsafe fn ddp_runtime_error(code: DDPInt, fmt: *const std::ffi::c_char);
}