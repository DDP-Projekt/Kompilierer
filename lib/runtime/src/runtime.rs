// TODO: signal handler
// TODO: LC_ALL/LC_NUMERIC windows vs linux
// TODO: init/end runtime

use debug_print::debug_println;
use libc::{SIGSEGV, signal};
use std::{
    ffi::{CStr, c_char, c_int},
    io::Write,
    sync::{
        Mutex,
        atomic::{AtomicBool, Ordering},
    },
};

use crate::ddptypes::{DDPList, DDPString};

unsafe extern "C" {
    pub fn ddp_runtime_error(code: i32, fmt: *const u8, ...) -> !;
}

#[cfg(target_os = "windows")]
unsafe extern "system" {
    fn SetConsoleCP(wCodePageID: u32) -> i32;
    fn SetConsoleOutputCP(wCodePageID: u32) -> i32;
}

extern "C" fn signal_handler(_sig: c_int) {
    debug_println!("caught SIGSEGV");
    unsafe {
        ddp_end_runtime();
        ddp_runtime_error(1, "Segmentation fault\n".as_ptr());
    }
}

unsafe extern "C" {
    fn setlocale(category: c_int, locale: *const u8) -> *mut c_char;
}

#[cfg(target_os = "windows")]
fn set_locales() {
    unsafe {
        setlocale(0, "German_Germany.utf8".as_ptr());
        setlocale(4, "Frensh_Canada.1252".as_ptr());
        SetConsoleCP(65001);
        SetConsoleOutputCP(65001);
    };
}

#[cfg(target_os = "linux")]
fn set_locales() {
    unsafe {
        setlocale(0, "de_DE.UTF-8".as_ptr());
    };
}

static mut CMD_ARGS: Mutex<Vec<DDPString>> = Mutex::new(Vec::new());

#[unsafe(no_mangle)]
pub extern "C" fn ddp_init_runtime(argc: c_int, argv: *const *const c_char) {
    debug_println!("\tinit_runtime\n");

    set_locales();

    unsafe {
        signal(SIGSEGV, signal_handler as usize);
    }

    unsafe {
        (*(&raw mut CMD_ARGS))
            .lock()
            .unwrap()
            .reserve_exact(argc as usize);
    }
    for i in 0..argc {
        unsafe {
            let arg_ptr = *argv.offset(i as isize);
            (*(&raw mut CMD_ARGS))
                .lock()
                .unwrap()
                .push(DDPString::from_raw_parts(
                    arg_ptr as *const u8,
                    CStr::from_ptr(arg_ptr).to_bytes().len(),
                ));
        }
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn ddp_end_runtime() {
    static ENDING: AtomicBool = AtomicBool::new(false);

    if ENDING.swap(true, Ordering::SeqCst) {
        return;
    }

    let _ = std::io::stdout().flush();
    let _ = std::io::stderr().flush();

    debug_println!("end_runtime");
    unsafe {
        (*(&raw mut CMD_ARGS)).lock().unwrap().clear();
    }
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn Befehlszeilenargumente(_ret: *mut DDPList<DDPString>) {}
