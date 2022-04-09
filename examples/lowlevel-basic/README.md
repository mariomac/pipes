# Example pipeline for the node API

The pipeline in the main file has two Start nodes that send the data to two destination Middle
nodes (`odds` and `evens`). From there, the data follows their own branches until they
are eventually joined in the `printer` Terminal node.

Output:

```
even number: 2
odd number: 847
odd number: 59
odd number: 81
odd number: 81
even number: 0
odd number: 3
odd number: 1
odd number: 887
even number: 4
```