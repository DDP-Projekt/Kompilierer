use crate::ddptypes::DDPString;
use sha2::{Digest, Sha256, Sha512};

#[unsafe(no_mangle)]
pub extern "C" fn SHA_256(x: DDPString) -> DDPString {
	DDPString::from(Sha256::digest(x.to_string()).as_slice())
}

#[unsafe(no_mangle)]
pub extern "C" fn SHA_512(x: DDPString) -> DDPString {
	DDPString::from(Sha512::digest(x.to_string()).as_slice())
}