Somehow simplified (?) otel library.

Goals:
* just one package for everything
  * that includes both "API" and "SDK" - it's all in the same package
* only HTTP OTLP exporters - no grpc, no other exporters
* remove features that Victoria* doesn't support
  * like exemplars/filters with metrics
* only slog bridge
* strictly no ENV variable reading, everything explicit
  * I hate env var magic, sorry
  * this makes initialization a bit annoying, but defaults should be exported (TODO)
* related to that - removing most Options/WithXXX patterns
  * except for new spans I guess
* as few dependencies as possible
  * specifically no grpc
* no global state
  * (for now, there is still a global debug logger; TODO)

Maybe goals?
* don't use interfaces when you can use structs

Non-goals:
* adherence to specifications
  * most of things I don't like about otel-go (env vars, global vars, multiple layers) is mandated by the specs!
* simplicity of code - it's mostly copied from OTLP (with copyrights intact)
  * however what I don't need is removed
* mergeability of code back from upstread
  * I do changes to remove the env variable reading
* tests
  * all tests are removed because I do far too many changes, and don't include test packages

The docs are all wrong now, including the go doc - I am not rewriting them right now

Should you use it? I don't know. I make it half to learn OTLP.

So far: traces and logs done, metrics not done. Example in `./example-trace`