# BUILDER
FROM golang:1.19-buster as builder
# install
RUN echo "deb http://apt.llvm.org/buster/ llvm-toolchain-buster-11 main" | tee /etc/apt/sources.list.d/llvm.list
RUN wget -O - https://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add -
RUN apt update && apt install -y clang-11 python3 python3-pip
RUN python3 -m pip install -U pip
RUN ln -s /usr/bin/clang-11 /usr/bin/clang


WORKDIR /src/vqlite
COPY . .

WORKDIR /src/vqlite/vqindex
RUN bash build.sh vqindex_api
RUN cp libs/*so* /usr/lib
RUN ln -s  /usr/lib/libtensorflow_framework.so.2 /usr/lib/libtensorflow_framework.so

WORKDIR /src/vqlite
RUN go build -o /go/bin/vqlite cmd/main.go


# OUTPUT
FROM debian:buster-slim
WORKDIR /app
COPY --from=BUILDER /usr/lib/libtensorflow_framework.so.2 /usr/lib/libtensorflow_framework.so.2
RUN ln -s  /usr/lib/libtensorflow_framework.so.2 /usr/lib/libtensorflow_framework.so
COPY --from=BUILDER /usr/lib/libvqindex_api.so /usr/lib/libvqindex_api.so
COPY --from=BUILDER /go/bin/vqlite /app/vqlite
#CMD ["/app/vqlite run"]
ENTRYPOINT ["/app/vqlite", "run"]