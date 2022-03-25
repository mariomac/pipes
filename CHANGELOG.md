# CHANGELOG

# v0.4.0

* Breaking changes:
    - `node.Init` renamed to `node.Starting`

# v0.3.0

* Inter-node communication input channels are now unbuffered by default. To make them buffered,
  you can append the `node.ChannelBufferLen` function to the `AsMiddle` and `AsTerminal` functions.

# v0.2.0

* Ported to Go 1.18 and generics for faster and safer execution.

# v0.1.1

* Added InType and OutType inspection functions to the Nodes

# v0.1.0

* Initial import from github.com/mariomac/go-pipes
