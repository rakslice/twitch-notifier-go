
package main

import (
	"testing"
	"github.com/jarcoal/httpmock"
)

// TEST CONTEXT TO PROVIDE CUSTOM ASSERT METHODS ON THE TEST CLASS

type TestContext struct {
	t *testing.T
}

func Ctx(t *testing.T) *TestContext {
	return &TestContext{t}
}

func (ctx *TestContext) assertGotErr(expectedMsg string, err error, desc string) {
	ctx.assert(err != nil, "In %s, expected error message '%s' but got no error", desc, expectedMsg)
	actualMessage := err.Error()
	ctx.assert(actualMessage == expectedMsg, "In %s, expected error messge '%s' but got error message '%s'", desc, expectedMsg, actualMessage)
}

func (ctx *TestContext) assertNoErr(err error, desc string) {
	if err != nil {
		ctx.assert(err == nil, "In %s, expected no error, but got error message '%s'", desc, err.Error())
	}
}

func (ctx *TestContext) assert(cond bool, fmt string, args... interface{}) {
	if !cond {
		ctx.t.Errorf(fmt, args...)
	}
}

// TESTS

func TestMissingFieldError(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai",
		httpmock.NewStringResponder(200, `{}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	ctx.assertGotErr("Response object was missing the 'somethings' field", err, "PagedKraken()")
	ctx.assert(pk == nil, "paged kraken was not nil even through constructor gave error")
}


func TestMissingTotalField(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai",
		httpmock.NewStringResponder(200, `{"somethings": []}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	ctx.assertGotErr("Response object was missing the '_total' field", err, "PagedKraken()")
	ctx.assert(pk == nil, "paged kraken was not nil even through constructor gave error")
}

func TestNoObjects(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai",
		httpmock.NewStringResponder(200, `{"somethings": [], "_total": 0}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	ctx.assertNoErr(err, "PagedKraken()")
	ctx.assert(pk != nil, "paged kraken was nil even through it should be a no-item iterator")

	ctx.assert(!pk.More(), "even through there are no items, pk.More() was true")
}

func TestTotalNonZeroButFirstPageEmpty(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai",
		httpmock.NewStringResponder(200, `{"somethings": [], "_total": 42}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	ctx.assertGotErr("Response object '_total' was 42 but the page was empty", err, "PagedKraken()")
	ctx.assert(pk == nil, "paged kraken was not nil even through constructor gave error")
}

func TestTotalNonZeroButLaterPageEmpty(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=0",
		httpmock.NewStringResponder(200, `{"somethings": ["meat popsicle"], "_total": 42}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	ctx.assertNoErr(err, "PagedKraken()")
	ctx.assert(pk != nil, "paged kraken was nil even through it should be a working iterator")

	var actualVal string

	ctx.assert(pk.More(), "expected more items at start but there are no more")

	nextErr := pk.Next(&actualVal)

	ctx.assertNoErr(nextErr, "first Next() call")
	ctx.assert(actualVal == "meat popsicle", "Next() did not produce the first iterator value")

	httpmock.Reset()

	ctx.assert(pk.More(), "expected more items after first page but there are no more")

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=1",
		httpmock.NewStringResponder(200, `{"somethings": [], "_total": 42}`))

	err = pk.Next(&actualVal)

	ctx.assertGotErr("Response object '_total' was 42 but the page was empty", err, "PagedKraken()")
}
