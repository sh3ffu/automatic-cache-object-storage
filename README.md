# Automatic Caching For Cloud Object Storage

A transparent proxy, also known as an inline proxy, intercepts client requests and redirects them without modifying the request or requiring client-side configuration. It operates invisibly to users, meaning they are unaware of its presence and do not need to adjust their network settings. This technology is used to analyse object storage requests and cache them for future use. 

In this repository, I implementated Transparent proxy using eBPF. Specifically, I utilize Golang alongside the ebpf-go package.

## Components

- **eBPF Program**: The eBPF program is responsible for intercepting and redirecting network traffic. It is written in C and compiled using LLVM. The program is loaded into the kernel using using the `bpf` system call.

- **Golang Program**: The Golang program is responsible for loading the eBPF program into the kernel and setting up the network interface. It uses the ebpf-go library to interact with the eBPF program.

## How to Run

### Using Makefile



To build and run the eBPF program, simply run the following command:
```
make run
```

To run the program, run the following command:
```
sudo ./build/proxy
```

### Using Go

First build and run the eBPF program:
```
go generate
go build
sudo ./proxy
```

Now let's verify it works as expected:

- Run the HTTP Server from `/test` directory `go run main.go`

- From another shell, run `curl http://localhost:8000`

You can then inspect eBPF logs using `sudo cat /sys/kernel/debug/tracing/trace_pipe` to verify transparent proxy indeed intercepts the network traffic.
