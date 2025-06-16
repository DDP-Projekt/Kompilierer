use rand;
use ddpruntime::ddptypes::*;

#[unsafe(no_mangle)]
pub extern "C" fn Zufalls_Kommazahl(a: DDPFloat, b: DDPFloat) -> DDPFloat {
	rand::random_range(a..=b)
}

#[unsafe(no_mangle)]
pub extern "C" fn Zufalls_Zahl(a: DDPInt, b: DDPInt) -> DDPInt {
	rand::random_range((a + 1)..=b)
}

#[unsafe(no_mangle)]
pub extern "C" fn Zufalls_Wahrheitswert(p: DDPFloat) -> DDPBool {
	rand::random_bool(p)
}
