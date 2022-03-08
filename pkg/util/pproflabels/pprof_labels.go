package pproflabels

import (
	"context"
	"net/http"
	"runtime/pprof"

	"github.com/opentracing/opentracing-go"
	"github.com/weaveworks/common/middleware"
	"github.com/weaveworks/common/tracing"
	"github.com/weaveworks/common/user"
	"google.golang.org/grpc"
)

// HTTPMiddleware adds request-specific pprof labels to the goroutine for the duration of the request.
func HTTPMiddleware() middleware.Interface {
	return middleware.Func(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			if sp := opentracing.SpanFromContext(ctx); sp != nil {
				traceID, ok := tracing.ExtractTraceID(r.Context())
				if ok {
					sp.SetTag("profile_id", traceID)
				}
			}
			pprof.Do(ctx, extractPprofLabels(ctx), func(ctx context.Context) {
				r = r.WithContext(ctx)
				handler.ServeHTTP(w, r)
			})
		})
	})
}

// GRPCMiddleware adds request-specific pprof labels to the goroutine for the duration of the request.
func GRPCMiddleware() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if sp := opentracing.SpanFromContext(ctx); sp != nil {
			traceID, ok := tracing.ExtractTraceID(ctx)
			if ok {
				sp.SetTag("profile_id", traceID)
			}
		}
		pprof.Do(ctx, extractPprofLabels(ctx), func(ctx context.Context) {
			resp, err = handler(ctx, req)
		})
		return
	}
}

// GRPCStreamMiddleware adds request-specific pprof labels to the goroutine for the duration of the request.
func GRPCStreamMiddleware() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx := ss.Context()
		if sp := opentracing.SpanFromContext(ctx); sp != nil {
			traceID, ok := tracing.ExtractTraceID(ctx)
			if ok {
				sp.SetTag("profile_id", traceID)
			}
		}
		pprof.Do(ctx, extractPprofLabels(ctx), func(ctx context.Context) {
			err = handler(srv, ss)
		})
		return
	}
}

func extractPprofLabels(ctx context.Context) (labels pprof.LabelSet) {
	traceID, _ := ExtractTraceID(ctx)
	orgID, _ := user.ExtractOrgID(ctx)
	return pprof.Labels("profile_id", traceID, "tenant", orgID)
}

// ExtractTraceID extracts the trace id, if any from the context.
func ExtractTraceID(ctx context.Context) (string, bool) {
	// Extract from OpenTracing Jaeger exporter
	traceID, ok := tracing.ExtractTraceID(ctx)
	if ok {
		return traceID, true
	}
	return "", false
}
