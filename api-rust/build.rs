fn main() {
    tonic_build::compile_protos("proto/weather.proto")
        .expect("Failed to compile protos");
}
