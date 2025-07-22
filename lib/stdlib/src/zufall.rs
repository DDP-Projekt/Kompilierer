use ddpruntime::ddptypes::*;
use rand::{self};

#[unsafe(no_mangle)]
pub extern "C" fn Zufalls_Kommazahl(a: DDPFloat, b: DDPFloat) -> DDPFloat {
    ((b - a) * rand::random_range(0.0..=1.0) + a) as DDPFloat
}

#[unsafe(no_mangle)]
pub extern "C" fn Zufalls_Zahl(a: DDPInt, b: DDPInt) -> DDPInt {
    rand::random_range((a + 1)..=b)
}

#[unsafe(no_mangle)]
pub extern "C" fn Zufalls_Wahrheitswert(p: DDPFloat) -> DDPBool {
    rand::random_bool(p)
}

#[cfg(test)]
mod tests {
    use core::f64;

    use crate::zufall::Zufalls_Kommazahl;

    #[test]
    fn test_finit() {
        // should not panic
        Zufalls_Kommazahl(f64::MIN, f64::MAX);
    }
}
