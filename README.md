# JSONDIFF

Based on: https://github.com/elgris/jsondiff

A simple little tool that produces readable diff of 2 JSON-able
(read "convertible to `map[string]interface{}`") objects. Useful
for diagnostics or debugging.

A command line interface that uses this is: https://github.com/pschlump/check-json-syntax
It also provides syntax checking for JSON files.

## Installation

```
go get github.com/elgris/jsondiff
```

## Examples of the output

<img width="508" alt="screen shot 2016-05-20 at 13 03 06" src="https://cloud.githubusercontent.com/assets/1905821/15427207/58a141e2-1e8b-11e6-8d99-c2d752a80699.png">

## Limitation

- Coloured output tested with terminals only.  This only works on Mac and Linux.  I will modify this so it works on windows in a few days.
- The tool converts input data into `map[string]interface{}` or `[]interface{}` with json encoding/decoding. Hence, types of input map will change during unmarshal step: integers become float64 and so on (check https://golang.org/pkg/encoding/json/ for details).   

## License

MIT
