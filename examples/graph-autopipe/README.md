# Example of user-configurable pipeline

This example demonstrates how to user the high-level API.

The `main` function in the `autopipe.go` file does the following:

* Registers three types of pipeline stages:
    - `HttpIngest`, which listens for HTTP POST connections and forwards them as a byte array.
    - `FieldDeleter`, which receives a map, deletes the fields specified in the configuration,
      and forwards it to the next stage of the pipeline.
    - `StdoutExport`, which receives strings and prints them in the standard output.

* Registers codecs that will automatically wire different types of stages, in case they have
  incompatible stages. For example, if you wire the output of the `HttpIngest` to the `StdoutExport`
  stage, you need a codec that transforms byte arrays to strings; if you wire the output of the
  `HttpIngest` to the `FieldDeleter` stage, you need a codec that transforms JSON byte arrays
  to maps.

* Reads the config that instantiates the different stages of the pipeline and connects them.
  The Graph Builder will automatically add the codecs in the different stages, if they have
  incompatible input/output types.

You can get a look to the `nodes.hcl` example configuration file, where an HTTP server forwards
its output to two stages: a `FieldDeleter` that removes any field named "password" and "secret",
and a `StdoutExport` that just prints the received messages. At the same time, the `FieldDeleter`
forwards the "safe" maps to another `StdoutExport`.

You can run the example with:

```
go run autopipe.go -graph nodes.hcl
```

In another terminal, you can submit an example JSON:

```
curl -X POST -d '{"hello":"my friend","password":"sup3rs3cr37","secret":"kadlfjjsdlaf"}' http://localhost:8080
```

In the program standard output, you will see:

```
Received message: {"hello":"my friend","password":"sup3rs3cr37","secret":"kadlfjjsdlaf"}
Safe-to-show message: {"hello":"my friend"}
```

You can give a try modifying the parameters and connections of the `nodes.hcl` file, even instantiating
new stages or removing them.