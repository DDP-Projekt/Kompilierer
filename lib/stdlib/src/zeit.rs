use std::time::Duration;
use chrono::{Datelike, Local, Timelike};

use crate::ddptypes::{DDPFloat, DDPInt, DDPString};

#[unsafe(no_mangle)]
pub extern "C" fn Zeit_Seit_Programmstart() -> DDPInt {
	0 // TODO
}

#[unsafe(no_mangle)]
pub extern "C" fn Zeit_Lokal(ret: *mut DDPString) {
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

	unsafe {
		ret.write(DDPString::from(time))
	}
}

#[unsafe(no_mangle)]
pub extern "C" fn Warte(seconds: DDPFloat) {
    let duration = Duration::from_secs_f64(seconds);
    std::thread::sleep(duration);
}