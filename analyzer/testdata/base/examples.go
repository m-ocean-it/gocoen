package base

import "pkg"

// #constructor[NewFoo]
type Foo struct{} // want Foo:"constructor is NewFoo"

// Baz is a struct.
//
// #constructor[BazInit]
type Baz struct { // want `Constructor "BazInit" does not exist in the same package`
}

func NewFoo() Foo {
	return Foo{}
}

func FooNotConstructedProperly() {
	f1 := Foo{}    // want `"Foo" must be constructed with "NewFoo"`
	var f2 Foo     // want `"Foo" must be constructed with "NewFoo"`
	var f3 *Foo    // want `"Foo" must be constructed with "NewFoo"`
	f4 := new(Foo) // want `"Foo" must be constructed with "NewFoo"`

	_, _, _, _ = f1, f2, f3, f4
}

func FooConstructedProperly() {
	f := NewFoo()
	_ = f
}

func BarNotConstructedProperly() {
	_ = pkg.Bar{} // want `"Bar" must be constructed with "ConstructB"`
}

func BarConstructedProperly() {
	_ = pkg.ConstructB()
}

// #constructor[NewFooBarBaz]
type FooBarBaz struct{} // want `Constructor "NewFooBarBaz" must be a function`

type NewFooBarBaz struct{}

// #constructor[andABottleOfRum]
type hohoho struct{} // want `Constructor "andABottleOfRum" does not return anything`

func andABottleOfRum() {}

// #constructor[notWhatever]
type whatever struct{} // want `Constructor "notWhatever" does not return the corresponding type`

func notWhatever() hohoho {
	return hohoho{}
}

// #constructor[newSomeStruct]
type someStruct struct{} // want someStruct:"constructor is newSomeStruct"

func newSomeStruct() (string, *someStruct) { return "", nil }

var someS = someStruct{} // want `"someStruct" must be constructed with "newSomeStruct"`

var _, someS2 = newSomeStruct()

// #constructor[NewSomeEnum]
type SomeEnum int // want SomeEnum:`constructor is NewSomeEnum`

func NewSomeEnum() SomeEnum {
	n := SomeEnum(1)

	return n
}

func InvalidSomeEnumInitialization() SomeEnum {
	// TODO: This case would be nice to catch.
	return SomeEnum(-1)

	// TODO: And that, too.
	// return -1
}
