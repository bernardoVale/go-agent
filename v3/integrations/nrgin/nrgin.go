// Copyright 2020 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// Package nrgin instruments https://github.com/gin-gonic/gin applications.
//
// Use this package to instrument inbound requests handled by a gin.Engine.
// Call nrgin.Middleware to get a gin.HandlerFunc which can be added to your
// application as a middleware:
//
//	router := gin.Default()
//	// Add the nrgin middleware before other middlewares or routes:
//	router.Use(nrgin.Middleware(app))
//
// Example: https://github.com/newrelic/go-agent/tree/master/v3/integrations/nrgin/example/main.go
package nrgin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/internal"
	newrelic "github.com/newrelic/go-agent/v3/newrelic"
)

func init() { internal.TrackUsage("integration", "framework", "gin", "v1") }

// headerResponseWriter gives the transaction access to response headers and the
// response code.
type headerResponseWriter struct{ w gin.ResponseWriter }

func (w *headerResponseWriter) Header() http.Header       { return w.w.Header() }
func (w *headerResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *headerResponseWriter) WriteHeader(int)           {}

var _ http.ResponseWriter = &headerResponseWriter{}

// replacementResponseWriter mimics the behavior of gin.ResponseWriter which
// buffers the response code rather than writing it when
// gin.ResponseWriter.WriteHeader is called.
type replacementResponseWriter struct {
	gin.ResponseWriter
	replacement http.ResponseWriter
	code        int
	written     bool
}

var _ gin.ResponseWriter = &replacementResponseWriter{}

func (w *replacementResponseWriter) flushHeader() {
	if !w.written {
		w.replacement.WriteHeader(w.code)
		w.written = true
	}
}

func (w *replacementResponseWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *replacementResponseWriter) Write(data []byte) (int, error) {
	w.flushHeader()
	return w.ResponseWriter.Write(data)
}

func (w *replacementResponseWriter) WriteString(s string) (int, error) {
	w.flushHeader()
	return w.ResponseWriter.WriteString(s)
}

func (w *replacementResponseWriter) WriteHeaderNow() {
	w.flushHeader()
	w.ResponseWriter.WriteHeaderNow()
}

// Context avoids making this package 1.7+ specific.
type Context interface {
	Value(key interface{}) interface{}
}

// Transaction returns the transaction stored inside the context, or nil if not
// found.
func Transaction(c Context) *newrelic.Transaction {
	if v := c.Value(internal.GinTransactionContextKey); nil != v {
		if txn, ok := v.(*newrelic.Transaction); ok {
			return txn
		}
	}
	if v := c.Value(internal.TransactionContextKey); nil != v {
		if txn, ok := v.(*newrelic.Transaction); ok {
			return txn
		}
	}
	return nil
}

type handlerNamer interface {
	HandlerName() string
}

func getName(c handlerNamer, useNewNames bool) string {
	if useNewNames {
		if fp, ok := c.(interface{ FullPath() string }); ok {
			return fp.FullPath()
		}
	}
	return c.HandlerName()
}

// Middleware creates a Gin middleware that instruments requests.
//
//	router := gin.Default()
//	// Add the nrgin middleware before other middlewares or routes:
//	router.Use(nrgin.Middleware(app))
//
// Gin v1.5.0 introduced the gin.Context.FullPath method which allows for much
// improved transaction naming.  This Middleware will use that
// gin.Context.FullPath if available and fall back to the original
// gin.Context.HandlerName if not.  If you are using Gin v1.5.0 and wish to
// continue using the old transaction names, use
// nrgin.MiddlewareHandlerTxnNames.
func Middleware(app *newrelic.Application) gin.HandlerFunc {
	return middleware(app, true, getTransactionNew)
}

// MiddlewareWithTransaction creates a Gin middleware that instruments
// requests allowing the client to optionally start the Transaction before
// this middleware. This functionally is particularly useful if you want
// to instrument middleware that executes before Gin.
//
//
//
//  router := gin.Default()
//  // Add the nrgin middleware before other middlewares or routes:
//	router.Use(nrgin.MiddlewareWithTransaction(app, "context-key-associated-with-your-transaction"))
//
// Gin v1.5.0 introduced the gin.Context.FullPath method which allows for much
// improved transaction naming.  This Middleware will use that
// gin.Context.FullPath if available and fall back to the original
// gin.Context.HandlerName if not.  If you are using Gin v1.5.0 and wish to
// continue using the old transaction names, use
// nrgin.MiddlewareHandlerTxnNames.
func MiddlewareWithTransaction(app *newrelic.Application, key string) gin.HandlerFunc {
	return middleware(app, true, getTransactionFromContext(app, key))
}


// MiddlewareHandlerTxnNames creates a Gin middleware that instruments
// requests.
//
//	router := gin.Default()
//	// Add the nrgin middleware before other middlewares or routes:
//	router.Use(nrgin.MiddlewareHandlerTxnNames(app))
//
// The use of gin.Context.HandlerName for naming transactions will be removed
// in a future release.  Available in Gin v1.5.0 and newer is the
// gin.Context.FullPath method which allows for much improved transaction
// names.  Use nrgin.Middleware to take full advantage of this new naming!
func MiddlewareHandlerTxnNames(app *newrelic.Application) gin.HandlerFunc {
	return middleware(app, false, getTransactionNew)
}


// getTransaction defines a contract to retrieve a newrelic Transaction from an http.Request
type getTransaction func(app *newrelic.Application, r *http.Request, name string) *newrelic.Transaction

// getTransactionNew simply creates a new transaction with the given name
func getTransactionNew(app *newrelic.Application, r *http.Request, name string) *newrelic.Transaction {
	txn := app.StartTransaction(name)
	txn.SetWebRequestHTTP(r)
	return txn
}

// getTransactionFromContext attempts to retrieve the transaction from http.Request context
// and fallback to the original behavior in case of errors.
func getTransactionFromContext(app *newrelic.Application, key string) getTransaction {
	return func(app *newrelic.Application, r *http.Request, name string) *newrelic.Transaction {
		var txn *newrelic.Transaction
		v := r.Context().Value(key)
		if v == nil {
			return getTransactionNew(app, r,  name)
		} else {
			if val, ok := v.(*newrelic.Transaction); ok {
				txn = val
				txn.SetName(name)
			} else {
				return getTransactionNew(app, r,  name)
			}
		}
		return txn
	}
}

func middleware(app *newrelic.Application, useNewNames bool, getTxn getTransaction) gin.HandlerFunc {
	return func(c *gin.Context) {
		if app != nil {
			name := c.Request.Method + " " + getName(c, useNewNames)

			w := &headerResponseWriter{w: c.Writer}

			txn := getTxn(app, c.Request, name)
			defer txn.End()

			repl := &replacementResponseWriter{
				ResponseWriter: c.Writer,
				replacement:    txn.SetWebResponse(w),
				code:           http.StatusOK,
			}
			c.Writer = repl
			defer repl.flushHeader()

			c.Set(internal.GinTransactionContextKey, txn)
		}
		c.Next()
	}
}
