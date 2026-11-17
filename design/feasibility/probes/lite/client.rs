// A browser Rust client. JS supplies one operation: send a GET to Lite's worker.
use std::slice;

#[link(wasm_import_module = "lite")]
unsafe extern "C" {
    fn get(ptr: *const u8, len: usize);
}

#[unsafe(no_mangle)]
pub extern "C" fn run() {
    let path = b"/party/data/t/films.json?sql=select+title+from+films&_shape=array";
    unsafe { get(path.as_ptr(), path.len()) };
}

// The harness copies the response into this bounded buffer before calling check.
const LIMIT: usize = 4096;
static mut RESPONSE: [u8; LIMIT] = [0; LIMIT];

#[unsafe(no_mangle)]
pub extern "C" fn buffer() -> *mut u8 {
    (&raw mut RESPONSE).cast()
}

#[unsafe(no_mangle)]
pub extern "C" fn check(status: u32, len: usize) -> bool {
    // JSON writers may escape the accented character. The JS harness also
    // checks the parsed title; this checks bytes received on the Rust side.
    let row = b"from 5 to 7";
    status == 200 && len <= LIMIT && unsafe { slice::from_raw_parts(buffer(), len) }
        .windows(row.len()).any(|v| v == row)
}
