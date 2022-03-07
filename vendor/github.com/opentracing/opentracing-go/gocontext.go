package opentracing

import (
	"context"
	"expvar"
	"runtime/pprof"

	"github.com/google/uuid"
)

type contextKey struct{}

var activeSpanKey = contextKey{}

// ContextWithSpan returns a new `context.Context` that holds a reference to
// the span. If span is nil, a new context without an active span is returned.
func ContextWithSpan(ctx context.Context, span Span) context.Context {
	if span != nil {
		if tracerWithHook, ok := span.Tracer().(TracerContextWithSpanExtension); ok {
			ctx = tracerWithHook.ContextWithSpanHook(ctx, span)
		}
	}
	return context.WithValue(ctx, activeSpanKey, span)
}

// SpanFromContext returns the `Span` previously associated with `ctx`, or
// `nil` if no such `Span` could be found.
//
// NOTE: context.Context != SpanContext: the former is Go's intra-process
// context propagation mechanism, and the latter houses OpenTracing's per-Span
// identity and baggage information.
func SpanFromContext(ctx context.Context) Span {
	val := ctx.Value(activeSpanKey)
	if sp, ok := val.(Span); ok {
		return sp
	}
	return nil
}

// StartSpanFromContext starts and returns a Span with `operationName`, using
// any Span found within `ctx` as a ChildOfRef. If no such parent could be
// found, StartSpanFromContext creates a root (parentless) Span.
//
// The second return value is a context.Context object built around the
// returned Span.
//
// Example usage:
//
//    SomeFunction(ctx context.Context, ...) {
//        sp, ctx := opentracing.StartSpanFromContext(ctx, "SomeFunction")
//        defer sp.Finish()
//        ...
//    }
func StartSpanFromContext(ctx context.Context, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	return StartSpanFromContextWithTracer(ctx, GlobalTracer(), operationName, opts...)
}

// StartSpanFromContextWithTracer starts and returns a span with `operationName`
// using  a span found within the context as a ChildOfRef. If that doesn't exist
// it creates a root span. It also returns a context.Context object built
// around the returned span.
//
// It's behavior is identical to StartSpanFromContext except that it takes an explicit
// tracer as opposed to using the global tracer.
func StartSpanFromContextWithTracer(ctx context.Context, tracer Tracer, operationName string, opts ...StartSpanOption) (Span, context.Context) {
	if parentSpan := SpanFromContext(ctx); parentSpan != nil {
		opts = append(opts, ChildOf(parentSpan.Context()))
	}
	id := uuid.New().String()
	opts = append(opts, Tag{Key: "span_profile_id", Value: id}, Tag{Key: "span_profile_url", Value: getProfileURL(id)})
	span := tracer.StartSpan(operationName, opts...)
	pprofSpan := &pprofLabelsSpan{Span: span}
	pprof.SetGoroutineLabels(pprof.WithLabels(ctx, pprof.Labels("span_profile_id", id)))
	pprofSpan.ctx = ContextWithSpan(ctx, pprofSpan)
	return pprofSpan, pprofSpan.ctx
}

func getProfileURL(id string) string {
	component := expvar.Get("component").(*expvar.String).Value()
	purl := expvar.Get("profile_url_prefix").(*expvar.String).Value()
	return purl + `?query=` + component + `.cpu%7Bspan_profile_id%3D%22` + id + `%22%7D`
}

type pprofLabelsSpan struct {
	Span
	ctx context.Context
}

func (s *pprofLabelsSpan) Finish() {
	pprof.SetGoroutineLabels(pprof.WithLabels(s.ctx, pprof.Labels("span_profile_id", "")))
	s.Span.Finish()
}
