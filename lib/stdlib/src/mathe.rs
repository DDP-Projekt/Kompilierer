use ddpruntime::ddptypes::*;

#[unsafe(no_mangle)]
pub extern "C" fn Sinus(x: DDPFloat) -> DDPFloat {
    x.sin()
}

#[unsafe(no_mangle)]
pub extern "C" fn Kosinus(x: DDPFloat) -> DDPFloat {
    x.cos()
}

#[unsafe(no_mangle)]
pub extern "C" fn Tangens(x: DDPFloat) -> DDPFloat {
    x.tan()
}

#[unsafe(no_mangle)]
pub extern "C" fn Arkussinus(x: DDPFloat) -> DDPFloat {
    x.asin()
}

#[unsafe(no_mangle)]
pub extern "C" fn Arkuskosinus(x: DDPFloat) -> DDPFloat {
    x.acos()
}

#[unsafe(no_mangle)]
pub extern "C" fn Arkustangens(x: DDPFloat) -> DDPFloat {
    x.atan()
}

#[unsafe(no_mangle)]
pub extern "C" fn Hyperbelsinus(x: DDPFloat) -> DDPFloat {
    x.sinh()
}

#[unsafe(no_mangle)]
pub extern "C" fn Hyperbelkosinus(x: DDPFloat) -> DDPFloat {
    x.cosh()
}

#[unsafe(no_mangle)]
pub extern "C" fn Hyperbeltangens(x: DDPFloat) -> DDPFloat {
    x.tanh()
}

#[unsafe(no_mangle)]
pub extern "C" fn Areahyperbelsinus(x: DDPFloat) -> DDPFloat {
    x.asinh()
}

#[unsafe(no_mangle)]
pub extern "C" fn Areahyperbelkosinus(x: DDPFloat) -> DDPFloat {
    x.acosh()
}

#[unsafe(no_mangle)]
pub extern "C" fn Areahyperbeltangens(x: DDPFloat) -> DDPFloat {
    x.atanh()
}

#[unsafe(no_mangle)]
pub extern "C" fn Winkel(x: DDPFloat, y: DDPFloat) -> DDPFloat {
    x.atan2(y)
}

#[unsafe(no_mangle)]
pub extern "C" fn Gausssche_Fehlerfunktion(x: DDPFloat) -> DDPFloat {
    // TODO: maybe implement, not in rusts stdlib. We could remove this function idk what the use it has
    unimplemented!()
}

#[unsafe(no_mangle)]
pub extern "C" fn Runden(x: DDPFloat, n: DDPInt) -> DDPFloat {
    let shft = 10_f64.powi(n as i32);
    (x * shft).round() / shft
}
