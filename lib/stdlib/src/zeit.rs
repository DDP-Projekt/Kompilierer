use chrono::{Datelike, Local, Timelike};
use ddpruntime::ddptypes::{DDPFloat, DDPInt, DDPString};
use std::time::Duration;

#[unsafe(no_mangle)]
pub extern "C" fn Zeit_Seit_Programmstart() -> DDPInt {
    unimplemented!()
}

#[unsafe(no_mangle)]
pub extern "C" fn Zeit_Lokal(ret: &mut DDPString) {
    let now = Local::now();
    let time = format!(
        "{:02}:{:02}:{:02} {:02}.{:02}.{:04}",
        now.hour(),
        now.minute(),
        now.second(),
        now.day(),
        now.month(),
        now.year()
    );

    *ret = DDPString::from(time)
}

#[unsafe(no_mangle)]
pub extern "C" fn Warte(seconds: DDPFloat) {
    let duration = Duration::from_secs_f64(seconds);
    std::thread::sleep(duration);
}
