
# problem

[![GoDoc](https://godoc.org/github.com/lpar/problem?status.svg)](https://godoc.org/github.com/lpar/problem)

This is an implementation of [RFC 7807](https://tools.ietf.org/html/rfc7807) for Go.

RFC 7807 attempts to standardize HTTP errors into a machine-readable format 
which is more detailed than the simple 3-digit HTTP errors. The format is extensible to 
support more complex error reporting, such as passing back multiple validation 
errors as a single response. At the core of the standard is the "problem detail" object.

The `ProblemDetail` object type defined in this code satisfies the Go `error` interface, which
means you can use it as an `error` return value from function calls. It also supports error
wrapping and unwrapping as per Go 1.13 and up.

## Why use this?

Frequently a web application `Handler`/`HandlerFunc` will have its functionality broken 
up into multiple sub-functions, any of which could fail for a variety of reasons. For example, you 
might have something like:

```
func HandlePut(w http.ResponseWriter, r *http.Request) {
  newrec, err := decodeRequest(r)
  if err != nil {
    http.Error(w, err.Error(), http.StatusBadRequest)
  }
  currec, err := loadRecord(id)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
  err := saveRecord(id, newrec)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
  err := writeChangeLog(currec, newrec)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}
```

It would be nice if a bad request could result in something more specific than `400 Bad Request`.
There are all sorts of useful HTTP status errors like `404 Not Found` (for a PUT to a URI that doesn't
correspond to a valid object), `401 Unauthorized` if the user's login timed out, `403 Forbidden` if they're not allowed 
to change that particular record, `413 Request Entity Too Large` if the PUT request is too big, and so on.
Similarly, we might like server-side problems to be more specific than `500 Internal Server Error` if possible.

You could have the handler examine the type of the returned error value, assuming the code that created the error defined
a custom type, but that will bloat up the handler code quickly.

Another possible approach would be to have the `decodeRequest` function issue the appropriate HTTP response -- but then you 
need to make sure every code execution path results in only one HTTP response being issued. If you've ever seen Go 
warn you `http: multiple response.WriteHeader calls`, you'll know that spreading `http.Error` throughout your codebase quickly
becomes a problem. 

With `ProblemDetails`, the appropriate HTTP error code, error name and detailed error message for a human can be encapsulated into the 
returned `error`, ready for the handler to issue using `problem.Write(w, err)`.

You can mix `ProblemDetails` error returns with other kinds of error return. A convenience function will report 
an error including JSON details if it's a `ProblemDetails`, or construct a default Internal Server Error detail object otherwise:

```
  err := someFunction()
  if err != nil {
    problem.MustReport(w, err)
    return
  }
```

## Usage

To construct and return errors via the method chaining / fluent API:
 
  1. Construct an error with `problem.New(httpstatus)`
  2. Add details using either:
     a. `.Errorf(fmtstr, ...)` (like `fmt.Errorf`) or
     b. `.WithDetail` / `.WithErr`
  3. Return error, or write using `.Write`

Or, use the non-fluent shortcut methods:

```
problem.Errorf(httpstatus, fmtstr, ...) 
// like fmt.Errorf but with an HTTP status code

problem.Error(w, msg, status)
// like http.Error
```

To handle errors:

``` 
if err := problem.Report(w, err); err != nil {
  // err wasn't a problem details object, deal with it as you like here
}
```

or:

```
problem.MustReport(w, err)
// uses StatusInternalError if err isn't a problem details object
```

## Example

Suppose I have a decodeRequest method which starts like this:

```
func decodeRequest(r *http.Request) (int64, error) {
	id := chi.URLParam(r, "id")
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, problem.New(http.StatusBadRequest).Errorf("can't parse ID: %w", err)
	}
...
```

This is called by a handler:

```
func GetLocation(w http.ResponseWriter, r *http.Request) {
	id, err := rest.decodeRequest(r)
	if err != nil {
		problem.MustWrite(w, err)
		return
	}
...
```

A GET with an invalid ID then results in this API response:

```
HTTP/1.1 400 Bad Request
Content-Type: application/problem+json

{
  "status": 400,
  "title": "Bad Request",
  "detail": "can't parse ID: strconv.ParseInt: parsing \"che3\": invalid syntax",
  "type": "https://httpstatuses.com/400"
}
```

