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

unsafe extern "C" {
    pub unsafe fn ddp_reallocate(ptr: *mut u8, old_size: usize, new_size: usize) -> *mut u8;
}