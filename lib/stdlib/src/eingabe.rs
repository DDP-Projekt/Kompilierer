use std::io::Read;

use ddpruntime::{
    ddptypes::{DDPBool, DDPChar},
    utf8::utf8_indicated_num_bytes,
};
use debug_print::debug_println;

#[unsafe(no_mangle)]
pub extern "C" fn extern_lies_buchstabe(war_eof: &mut DDPBool) -> DDPChar {
    debug_println!("extern_lies_buchstabe");
    let mut buff = [0u8; 4];
    let result = if let Ok(_) = std::io::stdin().read(&mut buff[..1]) {
        let n = utf8_indicated_num_bytes(buff[0]) as usize;
        debug_println!("{n} bytes indicated");
        if n > 1 {
            debug_println!("reading more bytes");
            if let Err(_) = std::io::stdin().read(&mut buff[1..n]) {
                debug_println!("returning 0");
                *war_eof = true;
                return 0;
            }
        }

        debug_println!("buff {buff:?}");
        std::str::from_utf8(&buff).unwrap().chars().nth(0).unwrap() as u32
    } else {
        debug_println!("returning 0 again");
        *war_eof = true;
        0
    };
    debug_println!("returning result {result} {}", unsafe {
        char::from_u32_unchecked(result)
    });
    result
}
