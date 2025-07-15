// TODO: signal handler
// TODO: LC_ALL/LC_NUMERIC windows vs linux
// TODO: init/end runtime

// use std::{ffi::{c_char, c_int}, sync::Mutex};

// unsafe extern "C" {
//     fn setlocale(category: c_int, locale: *const u8) -> *mut c_char;
// }

// #[cfg(target_os = "windows")]
// fn set_locales() {
//     unsafe {
//         setlocale(0, "German_Germany.utf8".as_ptr());
//         setlocale(4, "Frensh_Canada.1252".as_ptr());
//     };
// }

// #[cfg(target_os = "linux")]
// fn set_locales() {
//     unsafe {
//         setlocale(0, "de_DE.UTF-8".as_ptr());
//     };
// }

// static cmd_args = Mutex::new(Vec<String>)

// #[unsafe(no_mangle)]
// pub unsafe extern "C" fn ddp_init_runtime(argc: c_int, argv: *const *const c_char) {
//     set_locales();
// }

unsafe extern "C" {
    pub fn ddp_runtime_error(code: i32, fmt: *const u8, ...);
}
