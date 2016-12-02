package main

/**
This is mostly straight out of https://github.com/Fugiman/kaet/blob/master/kraken.go
 */

import (
	"encoding/json"
	//"fmt"
	//"log"
	"net/http"
	//"net/url"
	"strings"
	//"sync"
	"time"
	"net/url"
	"strconv"
)

type Kraken struct {
	extraHeaders map[string]string
}

func InitKraken() *Kraken {
	out := &Kraken{}
	out.extraHeaders = make(map[string]string)
	return out
}

func (obj *Kraken) addHeader(headerName string, headerValue string) {
	obj.extraHeaders[headerName] = headerValue
}

type KrakenPager struct {
	krakenInstance *Kraken
	path []string
	resultsListKey string
	pageSize uint
	pageOffset uint

	endOfResults bool

	responseTotalFieldValue uint
	gotResponseTotalFieldValue bool

	currentPageInProgress bool
	currentPageDecoder *json.Decoder
	currentPageResponse *http.Response

	baseParams url.Values
}

func (state *KrakenPager) AddParam(key, value string) {
	state.baseParams.Add(key, value)
}

// More indicates whether there are additional results available via Next()
func (state *KrakenPager) More() bool {
	return !state.endOfResults
}

// This deserializes the next list item into val, with the same behavior as json.Unmarshal.
// New pages will be requested as necessary, so this may block until a request completes.
func (state *KrakenPager) Next(val interface{}) error {
	assert(!state.endOfResults, "already ended")
	if !state.currentPageInProgress {
		pageErr := state.loadPage()
		if pageErr != nil {
			return pageErr
		}

	}

	dec := state.currentPageDecoder

	state.assertWithCleanup(dec.More(), "page did not contain at least one item")

	err := dec.Decode(val)
	if (err != nil) {
		return nil
	}

	if !dec.More() {
		// we are at the end of the array for this page

		if !state.gotResponseTotalFieldValue {
			// still need to know the total number of items on the page

			// eat the array end
			arrayEnd, arrayEndTokenErr := dec.Token()
			state.assertWithCleanup(arrayEndTokenErr == nil, "json array start token error: %s", arrayEndTokenErr)
			arrayEndDelim, wasDelim := arrayEnd.(json.Delim)
			state.assertWithCleanup(wasDelim, "json array end token was not a delim, was %s in %s", arrayEnd, state.path)
			state.assertWithCleanup(arrayEndDelim == ']', "value for %s was not an array, was %s in %s",
				state.resultsListKey, arrayEndDelim, state.path)

			state.seekToResultsListArrayOrEnd()
		}

		assert(state.gotResponseTotalFieldValue, "didn't get a _total value in page")
		totalNumItems := state.responseTotalFieldValue

		state.cleanupPage()

		// figure out if there is another page
		endOfThisPage := state.pageOffset + state.pageSize
		if endOfThisPage >= totalNumItems {
			// nope, no more pages
			state.endOfResults = true
		} else {
			// more pages

			// get ready for the next page
			state.pageOffset += state.pageSize
		}
	}

	return nil
}

func (state *KrakenPager) seekToResultsListArrayOrEnd() {
	// iterate through the dictionary contents and stop until we get to the arg with the results list or the end
	dec := state.currentPageDecoder

	assert(state.currentPageInProgress, "Page was not in progress!")

	for dec.More() {
		t, tokenErr := dec.Token()
		state.assertWithCleanup(tokenErr == nil, "json dict entry token error in %s: %s", state.path, tokenErr)

		switch key := t.(type) {
		default:
			state.assertWithCleanup(false, "unexpected key type %T", t)
		case json.Delim:
			state.assertWithCleanup(key == '}', "unexpected delimiter %s", key)
			break
		case string:
			msg("pagedKraken processing key %s", key)
			if key == "_total" {
				state.assertWithCleanup(state.gotResponseTotalFieldValue == false, "duplicate total field. huh?")
				decodeError := dec.Decode(&state.responseTotalFieldValue)
				state.assertWithCleanup(decodeError == nil, "error getting total value")
				state.gotResponseTotalFieldValue = true
				msg("saved a total value %v", state.responseTotalFieldValue)
			} else if key == state.resultsListKey {
				// ok we're up to the results list we want... this should be an array
				arrayStart, arrayStartTokenErr := dec.Token()
				state.assertWithCleanup(arrayStartTokenErr == nil, "json array start token error: %s", arrayStartTokenErr)
				arrayStartDelim, wasDelim := arrayStart.(json.Delim)
				state.assertWithCleanup(wasDelim, "json array start token was not a delim, was %s in %s", arrayStart, state.path)
				state.assertWithCleanup(arrayStartDelim == '[', "value for %s was not an array, was %s in %s",
					state.resultsListKey, arrayStartDelim, state.path)
				// ok, next up is an array element ready to read
				return
			} else {
				// just eat the other values
				var unused interface{}
				decodeError := dec.Decode(&unused)
				state.assertWithCleanup(decodeError == nil, "error eating another value: %s", decodeError)
			}

		}
	}

	// if we got here we reached the end of the page
	state.cleanupPage()
}

func (pagerState *KrakenPager) cleanupPage() {
	if pagerState.currentPageInProgress{
		if pagerState.currentPageResponse == nil {
			msg("KrakenPager.cleanup(): currentPageResponse was already nil")
		} else {
			pagerState.currentPageResponse.Body.Close()
		}
		pagerState.currentPageResponse = nil
		pagerState.currentPageDecoder = nil
		pagerState.currentPageInProgress = false
	}
}

func (pagerState *KrakenPager) assertWithCleanup(condition bool, format string, args ...interface{}) {
	if !condition {
		pagerState.cleanupPage()
	}
	assert(condition, format, args...)
}

func (state *KrakenPager) loadPage() error {
	params := state.baseParams
	params.Add("limit", strconv.Itoa(int(state.pageSize)))
	params.Add("offset", strconv.Itoa(int(state.pageOffset)))

	msg("pagedKraken for %s loading entries %v to %v", state.path, state.pageOffset, state.pageOffset + state.pageSize)

	resp, err := state.krakenInstance.doAPIRequest(&params, state.path)
	if err != nil {
		return err
	}
	state.currentPageInProgress = true
	state.currentPageResponse = resp
	state.gotResponseTotalFieldValue = false

	dec := json.NewDecoder(resp.Body)
	state.currentPageDecoder = dec

	// read open bracket
	t, tokenErr := dec.Token()
	state.assertWithCleanup(tokenErr == nil, "json opening token error in %s: %s", state.path, tokenErr)
	tDelim, wasDelim := t.(json.Delim)
	state.assertWithCleanup(wasDelim, "json opening token was not a delim, was %s in %s", t, state.path)

	state.assertWithCleanup(tDelim == '{', "response was not an object")

	state.responseTotalFieldValue = 0
	state.gotResponseTotalFieldValue = false

	state.seekToResultsListArrayOrEnd()
	return nil
}

// PagedKraken is an iterator for calling APIs that provide lists of items using page semantics.
//
// Use this with APIs that take offset and limit GET parameters and
// respond with a JSON object that has a _total field and an arbitrarily-named field with an
// array with a non-zero number of items.
//
//    err, iter := PagedKraken("array_field_name", 25, "some", "path", "parts", "go", "here")
//    if err != nil {
//
//
func (obj *Kraken) PagedKraken(resultsListKey string, pageSize uint, path ...string) (*KrakenPager, error) {

	out := &KrakenPager{}

	out.krakenInstance = obj
	out.path = path
	out.resultsListKey = resultsListKey
	out.pageSize = pageSize
	out.baseParams = url.Values{}

	out.endOfResults = false

	// load the first page
	err := out.loadPage()
	if err != nil {
		// there was a problem
		return nil, err
	} else {
		// all good
		return out, nil
	}
}

func (obj *Kraken) doAPIRequest(params *url.Values, path []string) (*http.Response, error) {
	curUrl := strings.Join(append([]string{"https://api.twitch.tv/kraken"}, path...), "/")
	if params != nil {
		curUrl += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", curUrl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Client-ID", CLIENT_ID)
	for headerName, headerValue := range obj.extraHeaders {
		req.Header.Add(headerName, headerValue)
	}

	resp, err := http.DefaultClient.Do(req)
	return resp, err
}

func (obj *Kraken) kraken(data interface{}, path ...string) error {
	resp, err := obj.doAPIRequest(nil, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	assert(resp.StatusCode == 200, "got status code %s", resp.StatusCode)

	return json.NewDecoder(resp.Body).Decode(data)
}

func roundToSeconds(d time.Duration) time.Duration {
	return ((d + time.Second/2) / time.Second) * time.Second
}
//
//func getUptime(channel string) string {
//	var data struct {
//		Stream *struct {
//			CreatedAt time.Time `json:"created_at"`
//		}
//	}
//	err := kraken(&data, "streams", channel)
//	if err != nil || data.Stream == nil {
//		log.Printf("getUptime=%v", err)
//		return fmt.Sprintf("%s is not online", channel)
//	}
//
//	// if t, err := time.Parse(time.RFC3339, u); err == nil {}
//	return roundToSeconds(time.Since(data.Stream.CreatedAt)).String()
//}
//
//func getGame(channel string, rating bool) string {
//	var data struct {
//		Game string
//	}
//	err := kraken(&data, "channels", channel)
//	if err != nil {
//		log.Printf("getGame=%v", err)
//		return "API is down"
//	}
//
//	//if rating {
//	//	return getRating(data.Game)
//	//}
//	return data.Game
//}

//var ratings = struct {
//	sync.Mutex
//	m map[string]string
//}{m: make(map[string]string)}
//
//func getRating(game string) string {
//	ratings.Lock()
//	defer ratings.Unlock()
//
//	if r, ok := ratings.m[game]; ok {
//		return r
//	}
//
//	q := url.Values{
//		"count": {"1"},
//		"game":  {game},
//	}
//	if req, err := http.NewRequest("GET", "https://videogamesrating.p.mashape.com/get.php?"+q.Encode(), nil); err == nil {
//		req.Header.Add("X-Mashape-Key", MASHAPE_KEY)
//		req.Header.Add("Accept", "application/json")
//		if resp, err := http.DefaultClient.Do(req); err == nil {
//			defer resp.Body.Close()
//			var data []map[string]interface{}
//			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil && len(data) > 0 {
//				if score, ok := data[0]["score"].(string); ok && score != "" {
//					r := fmt.Sprintf("%s [Rating: %s]", game, score)
//					ratings.m[game] = r
//					return r
//				}
//			} else {
//				log.Print(err)
//			}
//		}
//	}
//
//	return game
//}
