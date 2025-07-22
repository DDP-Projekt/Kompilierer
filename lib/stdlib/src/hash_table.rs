use std::hash::{Hash, Hasher};

use ddpruntime::ddptypes::{DDPFloat, DDPInt, DDPString};

#[unsafe(no_mangle)]
pub extern "C" fn FNV_Hash(str: &DDPString) -> DDPInt {
    let mut s = std::hash::DefaultHasher::new();
    str.hash(&mut s);
    s.finish() as DDPInt
}

#[unsafe(no_mangle)]
pub extern "C" fn Kommazahl_Hash(k: DDPFloat) -> DDPInt {
    k.to_bits().cast_signed()
}
