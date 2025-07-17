use ddpruntime::ddptypes::DDPString;
use sha2::{Digest, Sha256, Sha512};

#[unsafe(no_mangle)]
pub extern "C" fn SHA_256(ret: &mut DDPString, x: &DDPString) {
    if x.is_empty() {
        *ret = DDPString::from("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");
    }

    let sha: String = Sha256::digest(x.to_string())
        .iter()
        .map(|f| format!("{:02x}", f))
        .collect();
    *ret = DDPString::from(sha)
}

#[unsafe(no_mangle)]
pub extern "C" fn SHA_512(ret: &mut DDPString, x: &DDPString) {
    if x.is_empty() {
        *ret = DDPString::from(
            "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
        );
    }

    let sha: String = Sha512::digest(x.to_string())
        .iter()
        .map(|f| format!("{:02x}", f))
        .collect();
    *ret = DDPString::from(sha)
}
