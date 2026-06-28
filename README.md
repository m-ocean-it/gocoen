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
````

Run the checks:
```
gocoen ./...
```

If there is initialization of a struct for which a constructor must be used, you'll get a warning like this: `"Foo" must be constructed with "NewFoo"`.

If the directive specifies a non-existing or in some way invalid constructor, you'll get a separate warning.

# Acknowledgements

Inspired by https://github.com/reflechant/constructor-check.
