# Graph API

* Allow multiple Middle and Terminal funcs, the same way we do with AsStart
* Graph node providers now can include an error in the signature
* Propagate context between providers (e.g. after the initialization of a first node, you can later
* Detect cycles (optional)
* Allow passing per-stage and per-instance options (e.b. buffer size for each concrete stage)
* Register: error if registering an existing configuration type. Suggest e.g using typedefs for same underlying type
* Instantiation: check if instanceID is duplicate
* optimization: if many destinations share the same codec, instantiate it only once
* `enabled` tag in a boolean field as alternative to Enabler interface.

# Node API
* Way of propagating errors