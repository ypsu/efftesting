# efftesting: effect testing

efftesting is a unit testing that manages test expectations automatically.
It's main feature is an `Expect(effectName string, want any, got string)` function.
It stringifies `want` and compares that string to `got` and fails the test if they are not equal.
The magic is this: if `got` is wrong, efftesting can automatically update the source code to the new value.
It should make updating the tests for a code's *effects* a bit easier.
`effectName` is just an arbitrary name to make the test log messages clearer.

Example:

```
func MyStringLength(s string) int {
  return len(s)
}

func TestLength(t *testing.T) {
  et := efftesting.ET(t)
  et.Expect("string length of tükör", MyStringLength("tükör"), "7")
}

func TestMain(m *testing.M) {
  os.Exit(efftesting.Main(m))
}
```

Suppose you change the string to count utf8 characters instead of bytes:

```
func MyStringLength(s string) int {
  return utf8.RuneCountInString(s)
}
```

The expectation now fails with this:

```
$ go test example_test.go
--- FAIL: TestLength (0.00s)
    example_test.go:17: Non-empty diff for effect "string length of tükör", diff (-want, +got):
        -7
        +5
FAIL
Expectations need updating, use `EFFTESTING_UPDATE=1 go test ./...` for that.
```

Rerun the test with the `EFFTESTING_UPDATE=1` envvar to update the test expectation to expect 5 if that was expected from the change.

There's also a `Check(effectName string, want any, got string)` variant that quits the test if the expectation doesn't match.
So instead of this:

```
...
foo, err = foolib.New()
if err != nil {
  t.Failf("foolib.New() failed: %v.", err)
}
...
```

it's possible to write this:

```
foo, err = foolib.New()
et.Check("foolib.New() succeeded", err, "null")
```

You don't need to know beforehand that `err`'s value will be stringified to `null`.
Initially add only `et.Check("foolib.New() succeeded", err, "")` and then simply run update expectation command as described above.
In fact you no longer need to know beforehand what any expression's result will be, you only need to tell if a result is correct or not.
Ideal for code whose result is more a matter of taste rather than correctness (e.g. markdown rendering).

Most typical expectations can be rewritten to efftesting expectations.
E.g. a `EXPECT_LESS_THAN(3, 4)` can be rewritten to `Expect("comparison", 3 < 4, "true")`.
`Expect` and `Check` can be a powerful building block to make managing tests simpler.
They are unconventional so might take a while to get the hang of them.

efftesting formats multiline strings with backticks.
For convenience it formats structs and slices into a multiline json:

```
  et.Expect("slice example", strings.Split("This is a sentence.", " "), `
    [
      "This",
      "is",
      "a",
      "sentence."
    ]`)
```

Tip: include the string "TODO" in effectName for expectations that are still under development and are thus incorrect.
This allows committing the expectations first.
Once the correct implementation is in, the tests can be quickly updated with a single command.
The only additional work then needed is removing the TODO markers while verifying the correctness of the expectations.
Makes a test driven development much easier.
