
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

func (ctx *TestContext) assertGotErr(expectedMsg string, err error, desc string) bool {
	if ctx.assert(err != nil, "In %s, expected error message '%s' but got no error", desc, expectedMsg) {return true}
	actualMessage := err.Error()
	return ctx.assert(actualMessage == expectedMsg, "In %s, expected error messge '%s' but got error message '%s'", desc, expectedMsg, actualMessage)
}

func (ctx *TestContext) assertNoErr(err error, desc string) bool {
	if err != nil {
		return ctx.assert(err == nil, "In %s, expected no error, but got error message '%s'", desc, err.Error())
	}
	return false
}

func (ctx *TestContext) assert(cond bool, fmt string, args... interface{}) bool {
	if !cond {
		ctx.t.Errorf(fmt, args...)
		return true
	} else {
		return false
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
	if ctx.assertGotErr("Response object was missing the 'somethings' field", err, "PagedKraken()") {return}
	if ctx.assert(pk == nil, "paged kraken was not nil even through constructor gave error") {return}
}


func TestMissingTotalField(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai",
		httpmock.NewStringResponder(200, `{"somethings": []}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	if ctx.assertGotErr("Response object was missing the '_total' field", err, "PagedKraken()") {return}
	if ctx.assert(pk == nil, "paged kraken was not nil even through constructor gave error") {return}
}

func TestNoObjects(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai",
		httpmock.NewStringResponder(200, `{"somethings": [], "_total": 0}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	if ctx.assertNoErr(err, "PagedKraken()") {return}
	if ctx.assert(pk != nil, "paged kraken was nil even through it should be a no-item iterator") {return}

	if ctx.assert(!pk.More(), "even through there are no items, pk.More() was true") {return}
}

func TestTotalNonZeroButFirstPageEmpty(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai",
		httpmock.NewStringResponder(200, `{"somethings": [], "_total": 42}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	if ctx.assertGotErr("Response object '_total' was 42 but the page was empty", err, "PagedKraken()") {return}
	if ctx.assert(pk == nil, "paged kraken was not nil even through constructor gave error") {return}
}

func TestTotalNonZeroButLaterPageEmpty(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=0",
		httpmock.NewStringResponder(200, `{"somethings": ["meat popsicle"], "_total": 42}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	if ctx.assertNoErr(err, "PagedKraken()") {return}
	if ctx.assert(pk != nil, "paged kraken was nil even through it should be a working iterator") {return}

	var actualVal string

	if ctx.assert(pk.More(), "expected more items at start but there are no more") {return}

	nextErr := pk.Next(&actualVal)

	if ctx.assertNoErr(nextErr, "first Next() call") {return}
	if ctx.assert(actualVal == "meat popsicle", "Next() did not produce the first iterator value") {return}

	httpmock.Reset()

	if ctx.assert(pk.More(), "expected more items after first page but there are no more") {return}

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=1",
		httpmock.NewStringResponder(200, `{"somethings": [], "_total": 42}`))

	err = pk.Next(&actualVal)

	if ctx.assertGotErr("Response object '_total' was 42 but the page was empty", err, "PagedKraken()") {return}
}

func TestHTTPErrorOnInitialPage(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=0",
		httpmock.NewStringResponder(500, `{"error": "something is wrong"}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	if ctx.assertGotErr("Got HTTP status code 500 during page request", err, "PagedKraken()") {return}

	krakenErr, wasKrakenErr := err.(*KrakenError)
	if ctx.assert(wasKrakenErr, "expected error for HTTP error status to be KrakenError") {return}
	if ctx.assert(krakenErr.statusCode == 500, "Got status code %v for HTTP Error 500", krakenErr.statusCode) {return}

	if ctx.assert(pk == nil, "paged kraken was not nil even though initial request failed") {return}
}

func TestHTTPErrorOnLaterPage(t *testing.T) {
	ctx := Ctx(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=0",
		httpmock.NewStringResponder(200, `{"somethings": ["first thing"], "_total": 2}`))

	kraken := InitKraken()

	pk, err := kraken.PagedKraken("somethings", 1, "ohai")
	if ctx.assertNoErr(err, "PagedKraken()") {return}
	if ctx.assert(pk != nil, "Got null PagedKraken even through there was no error") {return}

	if ctx.assert(pk.More(), "No more items but we expected first item") {return}
	var value string
	nextError := pk.Next(&value)
	if ctx.assertNoErr(nextError, "PagedKraken.Next()") {return}
	if ctx.assert(value == "first thing", "did not get expected value from next; got %s", value) {return}

	httpmock.Reset()

	if ctx.assert(pk.More(), "No more items but we expected second item") {return}

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=1",
		httpmock.NewStringResponder(500, `{"error": "something is wrong"}`))

	nextError = pk.Next(&value)

	if ctx.assertGotErr("Got HTTP status code 500 during page request", nextError, "PagedKraken.Next()") {return}

	krakenErr, wasKrakenErr := nextError.(*KrakenError)
	if ctx.assert(wasKrakenErr, "expected error for HTTP error status to be KrakenError") {return}
	if ctx.assert(krakenErr.statusCode == 500, "Got status code %v for HTTP Error 500", krakenErr.statusCode) {return}

	// should be ready for retry

	httpmock.Reset()

	if ctx.assert(pk.More(), "No more items but we expected second item") {return}

	httpmock.RegisterResponder("GET", "https://api.twitch.tv/kraken/ohai?limit=1&offset=1",
		httpmock.NewStringResponder(200, `{"somethings": ["second thing"], "_total": 2}`))

	nextError = pk.Next(&value)
	if ctx.assertNoErr(nextError, "PagedKraken.Next()") {return}
	if ctx.assert(value == "second thing", "did not get expected value from next; got %s", value) {return}
}
