use ddpruntime::ddptypes::DDPString;
use typed_path::WindowsPath;

#[unsafe(no_mangle)]
pub extern "C" fn Windows_Saeubern(ret: &mut DDPString, pfad: &DDPString) {
    let pfad_string = pfad.to_string();
    let normalized_path = WindowsPath::new(pfad_string.as_str()).normalize();

    *ret = DDPString::from(normalized_path.as_bytes())
}

#[unsafe(no_mangle)]
pub extern "C" fn Windows_Pfad_Verbinden(ret: &mut DDPString, p1: &DDPString, p2: &DDPString) {
    let pfad1_string = p1.to_string();
    let pfad2_string = p2.to_string();

    let joined = WindowsPath::new(pfad1_string.as_str())
        .join(WindowsPath::new(pfad2_string.as_str()))
        .normalize();
    *ret = DDPString::from(joined.as_bytes())
}
