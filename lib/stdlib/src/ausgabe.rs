use ddpruntime::ddptypes::*;

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Zahl(x: DDPInt) {
    print!("{x}");
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Kommazahl(x: DDPFloat) {
    if x.is_infinite() {
        print!("{}Unendlich", if x.is_sign_positive() { "" } else { "-" })
    } else if x.is_nan() {
        print!("Keine Zahl (NaN)")
    } else {
        print!("{x}")
    }
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Byte(x: DDPByte) {
    print!("{x}");
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Wahrheitswert(x: DDPBool) {
    print!("{}", if x { "wahr" } else { "falsch" });
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Buchstabe(x: DDPChar) {
    print!("{}", char::from_u32(x).unwrap());
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Text(x: &DDPString) {
    print!("{}", x);
}

#[unsafe(no_mangle)]
pub extern "C" fn Schreibe_Fehler(x: &DDPString) {
    eprintln!("{}", x);
}
