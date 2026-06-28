# `gocoen` (Golang Constructor Enforcer)

Enforce that certain structs are initialized only via a specified constructor.

# Installation

```
go install github.com/m-ocean-it/gocoen@latest
```

# Usage

Add a directive like this:
```go
// #constructor[NewFoo]
type Foo struct{}

func NewFoo() *Foo { /* */ }
````

The directive may be surrounded by other text:
```go
// Foo is a great struct.
// Please, use #constructor[MakeFoo] for initialization.
// 
// Some more text.
type Foo struct {}

func MakeFoo() Foo { /* */ }
```

Multiple constructors may be specified and types other than structs may be used:
```go
//#constructor[NewMyInt1, NewMyInt2]
type MyInt int

func NewMyInt1() MyInt { /* */ }
func NewMyInt2() MyInt { /* */ }
````

Run the checks:
```
gocoen ./...
```

If there is initialization of a type for which a constructor must be used, you'll get a warning like this: `"Foo" must be constructed with "NewFoo"`.

If the directive specifies a non-existing or in some way invalid constructor, you'll get a separate warning.

# Acknowledgements

Inspired by https://github.com/reflechant/constructor-check.
