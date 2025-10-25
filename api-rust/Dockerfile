FROM rust:1.90 as builder

WORKDIR /app
COPY . .

RUN apt-get update && apt-get install -y protobuf-compiler
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates
COPY --from=builder /app/target/release/api_rust /usr/local/bin/

EXPOSE 8080
CMD ["api_rust"]
