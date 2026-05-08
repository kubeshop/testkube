# Testkube Expression Engine

JavaScript-like expression language for dynamic configuration and test workflow orchestration. Supports partial resolution when not all variables are available.

## Key Concepts

- **Expression**: A compiled piece of code that can be evaluated (e.g., `"name + ' is ' + string(age)"`)
- **StaticValue**: A concrete value that expressions resolve to (string, number, boolean, array, object)
- **Machine**: Execution context that provides variables and functions to expressions
- **Partial Resolution**: Ability to partially evaluate expressions when some variables are missing

## Basic Syntax

### Data Types
```
"string" 'string'        // Strings (single or double quotes)
123 123.45              // Numbers (int64, float64)
true false              // Booleans
[1, 2, 3]              // Arrays
{"a": 1, "b": 2}       // Objects
```

### Operators (by precedence)
```
!expr -expr             // Logical NOT, negation
* /                     // Multiply, divide
+ -                     // Add, subtract  
== != < <= > >=         // Comparison
&&                      // Logical AND
||                      // Logical OR  
expr ? true : false     // Ternary conditional
(expr)                  // Grouping
```

### Variable Access
```
variable                // Simple variable lookup
env.HOME               // Nested property access
config["key"]          // Bracket notation
array[0]               // Array indexing
```

### Templates
Embed expressions in strings using `{{}}` syntax:
```
"Hello {{name}}, status: {{passed ? 'OK' : 'FAIL'}}"
"Found {{length(items)}} item{{length(items) != 1 ? 's' : ''}}"
```

## Standard Library

### Type Conversion
| Function        | Description                                |
|-----------------|--------------------------------------------|
| `string(value)` | Convert any value to string                |
| `int(value)`    | Convert to integer                         |
| `float(value)`  | Convert to float                           |
| `bool(value)`   | Convert to boolean (JavaScript truthiness) |

### String Operations
| Function              | Description                                      |
|-----------------------|--------------------------------------------------|
| `trim(str)`           | Remove leading/trailing whitespace               |
| `split(str, sep?)`    | Split string into array (default separator: ",") |
| `join(array, sep?)`   | Join array into string (default separator: ",")  |
| `shellquote(args...)` | Quote arguments for shell safety                 |
| `shellparse(str)`     | Parse shell command into array                   |

### Collections
| Function                   | Description                         |
|----------------------------|-------------------------------------|
| `list(items...)`           | Create array from arguments         |
| `len(collection)`          | Get length of string, array, or map |
| `at(collection, index)`    | Get element at index/key            |
| `slice(array, start, end)` | Extract sub-array                   |
| `chunk(array, size)`       | Split array into chunks             |
| `range(start, end?)`       | Generate number range               |
| `filter(array, expr)`      | Filter elements by expression       |
| `map(array, expr)`         | Transform elements with expression  |
| `entries(map)`             | Convert map to [{key, value}] array |

### Math
| Function     | Description              |
|--------------|--------------------------|
| `floor(num)` | Round down to integer    |
| `ceil(num)`  | Round up to integer      |
| `round(num)` | Round to nearest integer |

### Data Formats
| Function          | Description            |
|-------------------|------------------------|
| `tojson(value)`   | Convert to JSON string |
| `json(str)`       | Parse JSON string      |
| `toyaml(value)`   | Convert to YAML string |
| `yaml(str)`       | Parse YAML string      |
| `jq(data, query)` | Apply jq query to data |

### Path Operations
| Function                  | Description       |
|---------------------------|-------------------|
| `abspath(path, base?)`    | Get absolute path |
| `relpath(dest, source?)`  | Get relative path |
| `makepath(parent, child)` | Join paths safely |

### Utility
| Function         | Description                                 |
|------------------|---------------------------------------------|
| `date(format?)`  | Current date (default: RFC3339 with millis) |
| `eval(expr)`     | Evaluate expression string                  |
| `any(values...)` | Return first non-null value                 |

### Filesystem (via libs.NewFsMachine)
| Function            | Description                 |
|---------------------|-----------------------------|
| `file(path)`        | Read file contents          |
| `glob(patterns...)` | Find files by glob patterns |

## Machine System

Machines provide the execution context for expressions by supplying variables and functions.

### Creating Machines
```go
// Basic machine
machine := expressions.NewMachine()

// Register simple variable
machine.Register("name", "John")

// Register nested data
machine.Register("config", map[string]interface{}{
    "timeout": 30,
    "enabled": true,
})

// Register dynamic accessor (e.g., for env.* variables)
machine.RegisterAccessor(func(name string) (interface{}, bool) {
    if strings.HasPrefix(name, "env.") {
        return os.Getenv(name[4:]), true
    }
    return nil, false
})

// Register custom function
machine.RegisterFunction("double", func(values ...expressions.StaticValue) (interface{}, bool, error) {
    if len(values) != 1 {
        return nil, false, errors.New("double requires 1 argument")
    }
    n, _ := values[0].IntValue()
    return n * 2, true, nil
})
```

### Combining Machines
```go
// Layer machines (right takes precedence)
combined := expressions.CombinedMachines(globalMachine, userMachine, requestMachine)
```

### Filesystem Machine
```go
// Adds file() and glob() functions
fsMachine := libs.NewFsMachine(os.DirFS("/"), "/workspace")
```

## Expression Compilation & Resolution

### Basic Usage
```go
// Compile expression
expr, err := expressions.Compile("name + ' is ' + string(age)")
if err != nil {
    // Handle syntax error
}

// Resolve with machine
result, err := expr.Resolve(machine)
if err != nil {
    // Handle runtime error
}

// Get final value
value := result.Value()
```

### Template Compilation
```go
template, err := expressions.CompileTemplate("Hello {{name}}!")
result, err := template.Resolve(machine)
```

### Partial Resolution
When not all variables are available:
```go
// Returns partially resolved expression instead of error
partial, changed, err := expr.SafeResolve(machine)
if changed {
    // Some variables were resolved
}
// Later, with more variables
final, err := partial.Resolve(fullMachine)
```

## Error Handling

```go
// Compile-time syntax errors
expr, err := expressions.Compile("invalid ++ syntax")

// Runtime resolution errors
result, err := expr.Resolve(machine)

// Safe resolution (no error for missing variables)
partial, changed, err := expr.SafeResolve(machine)
```

## Common Patterns

```go
// Default values
"value || 'default'"

// Nil-safe access
"user && user.email || 'no-email'"

// Conditional formatting
"status == 'passed' ? '✓' : '✗'"

// Environment variables with defaults
"env('PORT', '8080')"

// String interpolation
"'User ' + name + ' has ' + string(count) + ' items'"

// Conditional pluralization
"string(count) + ' item' + (count != 1 ? 's' : '')"
```
