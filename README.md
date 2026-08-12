Somehow simplified (?) otel library.

The principles:
* just one package for everything
* only HTTP OTLP exporters - no grpc, no other exporters
* strictly no ENV variable reading, everything explicit
  * I hate env var magic, sorry
* as few dependencies as possible
  * specifically no grpc


Non-goals:
* simplicity of code - it's mostly copied from OTLP (with copyrights intact)
* mergeability of code back from upstread
  * I do changes to remove the env variable reading
* tests
  * all tests are removed because I do far too many changes, and don't include test packages

Should you use it? I don't know. I make it half to learn OTLP.

So far: traces and logs done, metrics not done. Example in `./example-trace`