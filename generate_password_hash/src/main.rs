use argon2::{Argon2, PasswordHasher, password_hash::SaltString};
use rand_core::OsRng;

/** setting.jsonのpasswordのためのハッシュを作成 */
fn main() {
    let password = std::env::args()
        .nth(1)
        .expect("usage: generate_password_hash <password>");

    let salt = SaltString::generate(&mut OsRng);

    let hash = Argon2::default()
        .hash_password(password.as_bytes(), &salt)
        .expect("hash generation failed")
        .to_string();

    println!("{hash}");
}
