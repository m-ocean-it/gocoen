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
````

Run the checks:
```
gocoen ./...
```


# Acknowledgements

Inspired by https://github.com/reflechant/constructor-check.
