package main

import "testing"


// TEST CONTEXT TO PROVIDE CUSTOM ASSERT METHODS ON THE TEST CLASS

type TestContext struct {
	t *testing.T
}

func NewTestCtx(t *testing.T) *TestContext {
	return &TestContext{t}
}

/** These assert* methods return _false_ if the assertion passed and _true_ if it failed. This
    convention is to support easy early exiting on an assertion failure:

     if assert(okay_condition) {return}
 */

func (ctx *TestContext) assertGotErr(expectedMsg string, err error, desc string) bool {
	if ctx.assert(err != nil, "In %s, expected error message '%s' but got no error", desc, expectedMsg) {
		return true
	}
	actualMessage := err.Error()
	return ctx.assert(actualMessage == expectedMsg, "In %s, expected error messge '%s' but got error message '%s'", desc, expectedMsg, actualMessage)
}

func (ctx *TestContext) assertNoErr(err error, desc string) bool {
	if err != nil {
		return ctx.assert(err == nil, "In %s, expected no error, but got error message '%s'", desc, err.Error())
	}
	return false
}

func (ctx *TestContext) assertStrEqual(expected string, actual string, desc string) bool {
	return ctx.assert(expected == actual, "In %s, expected string '%s' but got '%s'", desc, expected, actual)
}

func (ctx *TestContext) assert(cond bool, fmt string, args ...interface{}) bool {
	if !cond {
		ctx.t.Errorf(fmt, args...)
		return true
	} else {
		return false
	}
}
