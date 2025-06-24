# Expressions Package

This document describes the expressions package, which provides a flexible and extensible expression evaluation system for Testkube.

> **User documentation:**<br>
> This is developer documentation for the Test Workflows expressions.<br>
> To read more about high-level usage, you can reach our [**official documentation**](https://docs.testkube.io/articles/test-workflows-expressions).

## Table of Contents

- [Why not existing language](#why-not-existing-language)
  - [Okay, but could it be done with known language?](#okay-but-could-it-be-done-with-known-language)
- [How expressions work](#how-expressions-work)
- [Expression Machines](#expression-machines)
  - [Creating a Machine](#creating-a-machine)
- [Adding new Expression functions](#adding-new-expression-functions)
  - [Example: Adding a custom function](#example-adding-a-custom-function)
  - [Example: Using RegisterFunctionExt for dynamic values](#example-using-registerfunctionext-for-dynamic-values)
  - [Adding to Standard Library](#adding-to-standard-library)
- [Simplify and Finalize](#simplify-and-finalize)
  - [Struct tags](#struct-tags)
  - [Simplify](#simplify)
  - [Finalize](#finalize)
  - [Force Variants](#force-variants)
- [Expression Types](#expression-types)

## Why not existing language

There are languages like [**cel-go**](https://github.com/google/cel-go) or [**tengo**](https://github.com/d5/tengo), or even sandboxed [**LUA**](https://github.com/Shopify/go-lua) or [**JS**](https://github.com/dop251/goja). Of course, they have advantage of external maintenance, but they are more complex than we need (inconvenient, but not a blocker) and **don't support partial resolution (blocker)**.

The partial resolution is needed, because part of the expression may need to be resolved before the TestWorkflow execution, but another part during the execution.

As an example, imagine a case:

```yaml
steps:
  - condition: "passed && config.enabled"
    shell: "npm test"
```

Now, while the `config.enabled` is available before the execution, the `passed` is a dynamic value that's not known until the step is executed.

With partial resolution, we will i.e. receive `condition: "passed"` when the `config.enabled=true` for the step, so the step itself doesn't need to know anything more than the actual dynamic context (`passed`).

### Okay, but could it be done with known language?

When using languages like CEL, we would need to prepare an optimization pipeline, that would:
* Parse CEL code to AST
* Find the nodes that could be replaced with their equivalent
* Conditionally try to extract them out of context and execute to replace
* Compile back to CEL

It would be basically kind of ahead of time compiling ([**AOT**](https://en.wikipedia.org/wiki/Ahead-of-time_compilation)), which would be just harder to implement and maintain, especially given that we would anyway need to have our own CEL interpreter (for AOT purpose).

Instead, we have a simple expressions language, built on top of JSON, that is built with partial resolution (instead of AOT) in mind, making it simple to work on.

> **Note:**
>
> Currently, given how extended this mechanism become, maybe it could be better to use CEL and write the layers above it.<br>
> It would still have both drawbacks and advantages over this mechanism, but at least the effort would be similar.
>
> At the point this was happening, it was faster to deliver own simple language instead of reusing existing one.
> It may happen, that in the future it will be just worth rewriting it to CEL under some feature flag.

## How expressions work

Expressions in Testkube are strings that can be evaluated to produce values. They support various operations, including:

- Variable access (e.g., `foo.bar`)
- Function calls (e.g., `string(123)`)
- Template strings (e.g., `"Hello, ${name}"`)

The expression system follows these steps:

1. **Parsing**: Expressions are parsed into an abstract syntax tree (AST)
2. **Resolution**: Expressions are resolved using one or more machines that provide context
3. **Evaluation**: The resolved expression produces a final value

Expressions can be used in various parts of Testkube, such as in test workflows, to dynamically compute values based on context.

## Expression Machines

Expression machines are the execution context for expressions. They provide:

1. **Variable Access**: Machines can provide values for variables used in expressions
2. **Function Execution**: Machines can execute functions called in expressions

The `Machine` interface defines two main methods:
- `Get(name string)`: Retrieves a variable by name
- `Call(name string, args []CallArgument)`: Calls a function with arguments

Machines can be chained together, allowing expressions to access variables and functions from multiple sources.

### Creating a Machine

```go
// Create a new machine
machine := expressions.NewMachine()

// Register a simple value
machine.Register("name", "John")

// Register a map of values
machine.RegisterMap("user", map[string]interface{}{
    "id": 123,
    "email": "john@example.com",
})

// Register a function
machine.RegisterFunction("greet", func(values ...expressions.StaticValue) (interface{}, bool, error) {
    name, _ := values[0].StringValue()
    return "Hello, " + name, true, nil
})

// Register a function that operate on non-resolved expression arguments, like `fn(unknown_variable)`
machine.RegisterFunctionExt("greet", func(args []expressions.CallArgument) (interface{}, bool, error) {
    name, _ := args[0].Expression.Template()
    return "Hello, " + name, true, nil
})
```

## Adding new Expression functions

To add a new expression function:

1. Create a handler function that processes arguments and returns a result
2. Register the function with a machine

### Example: Adding a custom function

```go
// Define a function handler for static values
func customFunction(values ...expressions.StaticValue) (interface{}, bool, error) {
    if len(values) != 2 {
        return nil, true, fmt.Errorf("function expects 2 arguments, %d provided", len(values))
    }

    arg1, _ := values[0].StringValue()
    arg2, _ := values[1].StringValue()

    return arg1 + "-" + arg2, true, nil
}

// Register the function with a machine
machine := expressions.NewMachine()
machine.RegisterFunction("customFunc", customFunction)
```

### Example: Using RegisterFunctionExt for dynamic values

The `RegisterFunctionExt` function allows you to work with expressions that might not be fully resolved yet. This is useful for:

1. Creating functions that need to inspect the raw expression structure
2. Implementing lazy evaluation or conditional resolution
3. Working with expressions that reference variables that might not exist yet

```go
// Register the function with a machine
machine := expressions.NewMachine()
machine.RegisterFunctionExt("dynamicFunc", dynamicFunction)

// Example usage with a conditional evaluator
machine.RegisterFunctionExt("ifResolved", func(args []expressions.CallArgument) (interface{}, bool, error) {
    if len(args) != 2 {
        return nil, true, fmt.Errorf("ifResolved expects 2 arguments, got %d", len(args))
    }

    // If first argument is resolved, return it, otherwise return second argument
    if args[0].Expression.Static() != nil {
        return args[0].Expression, true, nil
    }

    return args[1].Expression, true, nil
})
```

### Adding to Standard Library

To add a function to the standard library, add it to the `stdFunctions` map in [**`stdlib.go`**](./stdlib.go):

```go
stdFunctions["newFunction"] = StdFunction{
    ReturnType: TypeString,
    Handler: ToStdFunctionHandler(func(value ...StaticValue) (Expression, error) {
        // Function implementation
        return NewValue(result), nil
    }),
}
```

You can also use the extended syntax, like:

```go
stdFunctions["any"] = StdFunction{
    Handler: func(args []CallArgument) (Expression, bool, error) {
        resolved := true
        // Iterate over all arguments provided
        for i := range args {
            value := args[i].Static()

            // If it's not resolved - ignore
			if value == nil {
                resolved = false
                continue
			}

            // If it's resolved and not uses spread (...["a", "b"]), return it
            if !args[i].Spread {
                return value, true, nil
            }

            // If it's resolved and uses spread, find all array entries from this value
            items, err := value.SliceValue()
            if err != nil {
				return nil, true, fmt.Errorf("spread operator (...) used against non-list parameter: %s", value)
            }

            // Return the first item from this argument list
            if len(items) > 0 {
                return NewValue(items[0]), true, nil
            }
        }

        // If there is no values, like "any()" or "any(...[])", return None
        if resolved {
            return None, true, nil
        }
        // Otherwise, if there are some values, but none resolved, ignore it yet
		return nil, false, nil
    },
}
```

## Simplify and Finalize

The [**`generic.go`**](./generic.go) file provides functionality to process expressions embedded in Go structs using struct tags.

### Struct tags

Fields in structs can be tagged with `expr` to indicate they contain expressions:

```go
type Config struct {
    Name string `expr:"template"`  // String with template expressions
    Command string `expr:"expression"`  // Full expression
    Data map[string]string `expr:"template,template"` // Map with template expressions in template keys
}
```

### Simplify

The `Simplify` function processes expressions in a struct but keeps them as expressions:

```go
config := &Config{
    Name: "Hello, ${user.name}",
    Command: "string(123)",
}

// Process expressions but keep them as expressions
err := expressions.Simplify(config, machine)
```

After `Simplify`:
- Expressions are parsed and validated
- Variables are resolved if possible
- The structure of expressions is preserved

### Finalize

The `Finalize` function fully resolves expressions to their final values:

```go
// Fully resolve expressions to their final values
err := expressions.Finalize(config, machine)
```

After `Finalize`:
- Expressions are fully resolved to their final values
- The result contains only concrete values, not expressions

### Force Variants

Both functions have "Force" variants that process all string fields regardless of tags:

- `SimplifyForce`: Processes all fields as if they had the `include/template` tag (if not specified otherwise in tags)
- `FinalizeForce`: Fully resolves all fields as if they had the `expr` tag (if not specified otherwise in tags)

## Expression Types

Expressions can produce different types of values:

- `TypeString`: String values
- `TypeBool`: Boolean values
- `TypeInt64`: Integer values
- `TypeFloat64`: Floating-point values

The type system helps ensure type safety when evaluating expressions. It's quite loose though (similarly to i.e. JS), to make it more user-friendly.
