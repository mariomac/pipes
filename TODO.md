# Graph API
* Allow defininf configuraiton properties as pointers
* Instance IDs are optional
* Don't have to specify the connections. Create e.g. a default connection behavior (order of definition? predefined in the type?)
* Detect cycles
* Allow passing per-stage and per-instance options (e.b. buffer size for each concrete stage)
* Register: error if registering an existing configuration type. Suggest e.g using typedefs for same underlying type
* Instantiation: check if instanceID is duplicate
* optimization: if many destinations share the same codec, instantiate it only once

# Node API
* Way of propagating errors