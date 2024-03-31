# Tutorial

## Low-Level API

1. [Basic nodes](a-lowlevel/01-nodes/)
2. [How node connections work](a-lowlevel/02-connections)
3. [Start and end of a graph execution](a-lowlevel/03-start-end)
3. [Demuxed outputs](a-lowlevel/04-demuxes)

## High-Level API

**DEPRECATED API**: it will be removed in future versions of
this library.

The high-level API relies internally on the low-level API, so
it is recommended to first follow the low-level API tutorials
for a better understanding of the underlying behavior.

1. [Basic nodes](b-highlevel/01-basic-nodes)
1. [Graph struct annotations](b-highlevel/02-annotations)
1. [Optional nodes](b-highlevel/03-optional)
2. [Demuxed outputs](b-highlevel/04-demuxes)
1. [High-Level API: dynamic nodes](b-highlevel/04-dynamic-nodes/)
1. [High-Level API: codecs](b-highlevel/05-codecs/)
1. graph building options
1. Dynamic identification and connection of nodes?

## Advanced patterns on the high-level API

7. Sharing configuration/status across all the nodes (wrappers)