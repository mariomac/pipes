# Graph API

* Allow multiple Middle and Terminal funcs, the same way we do with AsStart and MultiStartProvider
* Detect cycles (optional)
* Allow passing per-stage and per-instance options (e.b. buffer size for each concrete stage)
* Register: error if registering an existing configuration type. Suggest e.g using typedefs for same underlying type
* Instantiation: check if instanceID is duplicate
* optimization: if many destinations share the same codec, instantiate it only once
* Don't force `Enabler` interface to be implemented as the same type of the struct field.
  Look for pointer and value receivers indistinctly.

# Node API
* Way of propagating errors