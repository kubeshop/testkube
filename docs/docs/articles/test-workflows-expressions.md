# Test Workflows - Expressions

## Expressions Language

We have designed a simple expressions language, that allows dynamic evaluation of different values.

## JSON-Native

It is built on JSON, so every JSON syntax is a valid expression value as well, like `[ "a", "b", "c" ]`.

## Math

You can do basic math easily, like **config.workers * 5**.

![Expressions](../img/expressions.png) 

### Operators

#### Arithmetic

The operators have the precedence defined so the order will follow math rules. Examples:

* `1 + 2 * 3` will result in `7`
* `(1 + 2) * 3` will result in `9`
* `2 * 3 ** 2` will result in `18`

| Operator                  | Returns                              | Description                                    | Example                                                                                                                                                                                                      |
|---------------------------|--------------------------------------|------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `==` (or `=`)             | `bool`                               | Is equal?                                      | `3 == 5` is `false`                                                                                                                                                                                          |
| `!=` (or `<>`)            | `bool`                               | Is not equal?                                  | `3 != 5` is `true`                                                                                                                                                                                           |
| `>`                       | `bool`                               | Is greater than?                               | `3 > 5` is `false`                                                                                                                                                                                           |
| `<`                       | `bool`                               | Is lower than?                                 | `3 < 5` is `true`                                                                                                                                                                                            |
| `>=`                      | `bool`                               | Is greater than or equal?                      | `3 >= 5` is `false`                                                                                                                                                                                          |
| `<=`                      | `bool`                               | Is lower than or equal?                        | `3 <= 5` is `true`                                                                                                                                                                                           |
| `&&`                      | the last value or the falsy one      | Are both truthy?                               | `true && false` is `false`<br/>`5 && 0 && 3` is `0`<br/>`5 && 3 && 2` is `2`                                                                                                                                 |
| <code>&#124;&#124;</code> | first truthy value or the last value | Is any truthy?                                 | <code>true &#124;&#124; false</code> is <code>true</code><br/><code>5 &#124;&#124; 3 &#124;&#124; 0</code> is `5`<br/><code>0 &#124;&#124; 5</code> is `5`<br/><code>"" &#124;&#124; "foo"</code> is `"foo"` |
| `!`                       | `bool`                               | Is the value falsy?                            | `!0` is `true`                                                                                                                                                                                               |
| `?` and `:`               | any of the values inside             | Ternary operator - if/else                     | `true ? 5 : 3` is `5`                                                                                                                                                                                        |
| `+`                       | `string` or `float`                  | Add numbers together or concatenate text       | `1 + 3` is `4`<br />`"foo" + "bar"` is `"foobar"`<br />`"foo" + 5` is `"foo5"`                                                                                                                               |
| `-`                       | `float`                              | Subtract one number from another               | `5 - 3` is `2`                                                                                                                                                                                               |
| `%`                       | `float`                              | Divides numbers and returns the remainder      | `5 % 3` is `2`                                                                                                                                                                                               |
| `/`                       | `float`                              | Divides two numbers                            | `6 / 3` is `2`<br />`10 / 4` is `2.5`<br />**Edge case:** `10 / 0` is `0` (for simplicity)                                                                                                                   |
| `*`                       | `float`                              | Multiplies one number by the other             | `4 * 2` is `8`                                                                                                                                                                                               |
| `**`                      | `float`                              | Exponentiation - power one number to the other | `2 ** 5` is `32`                                                                                                                                                                                             |
| `(` and `)`               | the inner type                       | Compute the expression altogether              | `(2 + 3) * 5` is `20`                                                                                                                                                                                        |

#### Access

| Operator | Description               | Example                                                                             |
|----------|---------------------------|-------------------------------------------------------------------------------------|
| `.`      | Access inner value        | `{"id": 10}.id` is `10`<br />`["a", "b"].1` is `"b"`                                |
| `.*.`    | Wildcard mapping          | `[{"id": 5}, {"id": 3}].*.id` is `[5, 3]`                                           |
| `...`    | Spread arguments operator | `shellquote(["foo", "bar baz"]...)` is equivalent of `shellquote("foo", "bar baz")` |

## Built-in Variables

### General Variables

There are some built-in variables available. Part of them may be resolved before execution (and therefore used for Pod settings),
while the others may be accessible only dynamically in the container.

#### Selected variables

| Name                                                       | Resolved immediately | Description                                                                       |
|------------------------------------------------------------|----------------------|-----------------------------------------------------------------------------------|
| `always`                                                   | ✅                    | Alias for `true`                                                                  |
| `never`                                                    | ✅                    | Alias for `false`                                                                 |
| `config` variables (like `config.abc`)                     | ✅                    | Values provided for the configuration                                             |
| `execution.id`                                             | ✅                    | TestWorkflow Execution's ID                                                       |
| `execution.name`                                           | ✅                    | TestWorkflow Execution's name                                                     |
| `execution.number`                                         | ✅                    | TestWorkflow Execution's sequence number                                          |
| `execution.scheduledAt`                                    | ✅                    | TestWorkflow Execution's scheduled at date                                        |
| `resource.id`                                              | ✅                    | Either execution ID, or unique ID for parallel steps and services                 |
| `resource.root`                                            | ✅                    | Either execution ID, or nested resource ID, of the resource that has scheduled it |
| `namespace`                                                | ✅                    | Namespace where the execution will be scheduled                                   |
| `workflow.name`                                            | ✅                    | Name of the executed TestWorkflow                                                 |
| `env` variables (like `env.SOME_VARIABLE`)                 | ❌                    | Environment variable value                                                        |
| `failed`                                                   | ❌                    | Is the TestWorkflow Execution failed already at this point?                       |
| `passed`                                                   | ❌                    | Is the TestWorkflow Execution still not failed at this point?                     |
| `services` (like `services.db.0.ip` or `services.db.*.ip`) | ❌                    | Get the IPs of initialized services                                               |

### Contextual Variables

In some contexts, there are additional variables available.

#### Retry Conditions

When using custom `retry` condition, you can use `self.passed` and `self.failed` for determining the status based on the step status.

```yaml
spec:
  steps:
  - shell: exit 0
    # ensure that the step won't fail for 5 executions
    retry:
      count: 5
      until: 'self.failed' 
```

#### Matrix and Shard

When using `services` (service pods), `parallel` (parallel workers), or `execute` (test suite) steps:

* You can use `matrix.<name>` and `shard.<name>` to access parameters for each copy.
* You can access `index` and `count` that will differ for each copy.
* Also, you may use `matrixIndex`, `matrixCount`, `shardIndex` and `shardCount` to get specific indexes/numbers for combinations and shards. 

```yaml
spec:
  services:
    # Start two workers and label them with index information
    db:
      count: 2
      description: "Instance {{ index + 1 }} of {{ count }}" # "Instance 1 of 2" and "Instance 2 of 2"
      image: mongo:latest
    # Run 2 servers with different node versions
    api:
      matrix:
        node: [20, 21]
      description: "Node v{{ matrix.node }}" # "Node v20" and "Node v21"
      image: "node:{{ matrix.node }}"
```

## Built-in Functions

### Casting

There are some functions that help to cast values to a different type. Additionally, when using wrong types in different places, the engine tries to cast them automatically.

| Name     | Returns                 | Description              | Example                                                                                                                   |
|----------|-------------------------|--------------------------|---------------------------------------------------------------------------------------------------------------------------|
| `string` | `string`                | Cast value to a string   | `string(5)` is `"5"`<br />`string([10, 15, 20])` is `"10,15,20"`<br />`string({ "foo": "bar" })` is `"{\"foo\":\"bar\"}"` |
| `list`   | list of provided values | Build a list of values   | `list(10, 20)` is `[ 10, 20 ]`                                                                                            |
| `int`    | `int`                   | Maps to integer          | `int(10.5)` is `10`<br />`int("300.50")` is `300`                                                                         |
| `bool`   | `bool`                  | Maps value to boolean    | `bool("")` is `false`<br />`bool("1239")` is `true`                                                                       |
| `float`  | `float`                 | Maps value to decimal    | `float("300.50")` is `300.5`                                                                                              |
| `eval`   | anything                | Evaluates the expression | `eval("4 * 5")` is `20`                                                                                                   |

### General

| Name         | Returns         | Description                                                                                                                                     | Example                                                                                                  |
|--------------|-----------------|-------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------|
| `join`       | `string`        | Join list elements                                                                                                                              | `join(["a", "b"])` is `"a,b"`<br />`join(["a", "b"], " - ")` is `"a - b"`                                |
| `split`      | `list`          | Split string to list                                                                                                                            | `split("a,b,c")` is `["a", "b", "c"]`<br />`split("a - b - c", " - ")` is `["a", "b", "c"]`              |
| `trim`       | `string`        | Trim whitespaces from the string                                                                                                                | `trim("   \nabc  d  ")` is `"abc  d"`                                                                    |
| `len`        | `int`           | Length of array, map or string                                                                                                                  | `len([ "a", "b" ])` is `2`<br />`len("foobar")` is `6`<br />`len({ "foo": "bar" })` is `1`               |
| `floor`      | `int`           | Round value down                                                                                                                                | `floor(10.5)` is `10`                                                                                    |
| `ceil`       | `int`           | Round value up                                                                                                                                  | `ceil(10.5)` is `11`                                                                                     |
| `round`      | `int`           | Round value to nearest integer                                                                                                                  | `round(10.5)` is `11`                                                                                    |
| `at`         | anything        | Get value of the element                                                                                                                        | `at([10, 2], 1)` is `2`<br />`at({"foo": "bar"}, "foo")` is `"bar"`                                      |
| `tojson`     | `string`        | Serialize value to JSON                                                                                                                         | `tojson({ "foo": "bar" })` is `"{\"foo\":\"bar\"}"`                                                      |
| `json`       | anything        | Parse the JSON                                                                                                                                  | `json("{\"foo\":\"bar\"}")` is `{ "foo": "bar" }`                                                        |
| `toyaml`     | `string`        | Serialize value to YAML                                                                                                                         | `toyaml({ "foo": "bar" })` is `"foo: bar\n`                                                              |
| `yaml`       | anything        | Parse the YAML                                                                                                                                  | `yaml("foo: bar")` is `{ "foo": "bar" }`                                                                 |
| `shellquote` | `string`        | Sanitize arguments for shell                                                                                                                    | `shellquote("foo bar")` is `"\"foo bar\""`<br />`shellquote("foo", "bar baz")` is `"foo \"bar baz\""`    |
| `shellparse` | `[]string`      | Parse shell arguments                                                                                                                           | `shellparse("foo bar")` is `["foo", "bar"]`<br />`shellparse("foo \"bar baz\"")` is `["foo", "bar baz"]` |
| `map`        | `list` or `map` | Map list or map values with expression; `_.value` and `_.index`/`_.key` are available                                                           | `map([1,2,3,4,5], "_.value * 2")` is `[2,4,6,8,10]`                                                      |
| `filter`     | `list`          | Filter list values with expression; `_.value` and `_.index` are available                                                                       | `filter([1,2,3,4,5], "_.value > 2")` is `[3,4,5]`                                                        |
| `jq`         | anything        | Execute [**jq**](https://en.wikipedia.org/wiki/Jq_(programming_language)) against value                                                         | <code>jq([1,2,3,4,5], ". &#124; max")</code> is `[5]`                                                    |
| `range`      | `[]int`         | Build range of numbers                                                                                                                          | `range(5, 10)` is `[5, 6, 7, 8, 9]`<br />`range(5)` is `[0, 1, 2, 3, 4]`                                 |
| `relpath`    | `string`        | Build relative path                                                                                                                             | `relpath("/a/b/c")` may be `./b/c`<br />`relpath("/a/b/c", "/a/b")` is `"./c"`                           |
| `abspath`    | `string`        | Build absolute path                                                                                                                             | `abspath("/a/b/c")` is `/a/b/c`<br />`abspath("b/c")` may be `/some/working/dir/b/c`                     |
| `chunk`      | `[]list`        | Split list to chunks of specified maximum size                                                                                                  | `chunk([1,2,3,4,5], 2)` is `[[1,2], [3,4], [5]]`                                                         |
| `date`       | `string`        | Return current date (either `2006-01-02T15:04:05.000Z07:00` format or custom argument ([**Go syntax**](https://go.dev/src/time/format.go#L101)) | `date()` may be `"2024-06-04T11:59:32.308Z"`<br />`date("2006-01-02")` may be `2024-06-04`               |

### File System

These functions are only executed during the execution.

| Name   | Returns    | Description           | Example                                                                                                          |
|--------|------------|-----------------------|------------------------------------------------------------------------------------------------------------------|
| `file` | `string`   | File contents         | `file("/etc/some/path")` may be `"some\ncontent"`                                                                |
| `glob` | `[]string` | Find files by pattern | `glob("/etc/**/*", "./x/**/*.js")` may be `["/etc/some/file", "/etc/other/file", "/some/working/dir/x/file.js"]` |

![Built-in Functions](../img/built-in-functions.png)
