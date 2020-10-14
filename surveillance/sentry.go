package surveillance

import (
	"github.com/Vernacular-ai/vcore/errors"
	"github.com/Vernacular-ai/vcore/log"
	sentryWrapper "github.com/Vernacular-ai/vcore/sentry"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/http"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"os"
)

type Sentry struct {
	client  *sentry.Client
	handler *sentryWrapper.Handler
}

func initSentry() (client *Sentry) {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn: dsn,
			// Use async transport. Which is set by default. Use Sync transport for testing.
			//Transport: sentry.NewHTTPSyncTransport(),

			// Enable debugging to check connectivity
			//Debug: true,
			Environment: os.Getenv("ENVIRONMENT"),
		}); err != nil {
			log.Warnf("Could not initialize sentry with DSN: %s", dsn)
			client = &Sentry{nil, nil}
		} else {
			client = &Sentry{
				sentry.CurrentHub().Client(),
				sentryWrapper.New(sentryhttp.Options{Repanic: true}),
			}
		}
	} else {
		log.Warnf("Could not initialize sentry with DSN: %s", dsn)
		client = &Sentry{nil, nil}
	}
	return
}

//func initSentry() *Sentry {
//	dsn := os.Getenv("SENTRY_DSN")
//	if dsn != "" {
//		if client, err := raven.New(dsn); err != nil {
//			return &Sentry{nil}
//		} else {
//			_client := &Sentry{client}
//			_client.client.SetRelease(os.Getenv("SENTRY_RELEASE"))
//			_client.client.SetEnvironment(os.Getenv("SENTRY_ENVIRONMENT"))
//			return _client
//		}
//	} else {
//		return &Sentry{nil}
//	}
//}

var (
	SentryClient = initSentry()
)

// Handles an error by capturing it on Sentry and logging the same on STDOUT
func (wrapper *Sentry) Capture(err error, _panic bool) {
	if err != nil {
		// Do not log to sentry if the error is ignorable.
		// However, do log it to stdout
		if wrapper.client != nil && !errors.Ignore(err) {
			// Capture error asynchronously
			sentry.WithScope(func(scope *sentry.Scope) {

				// Setting the stacktrace of the error as an extra along with any other extras set in the error
				if extras := errors.Extras(err); extras != nil{
					if abortLog, ok := extras["custom_service"]; ok && abortLog.(bool) {
						return
					}
					scope.SetExtras(extras)
					scope.SetExtra("stacktrace", errors.Stacktrace(err))
				} else {
					scope.SetExtras(map[string]interface{}{
						"stacktrace": errors.Stacktrace(err),
					})
				}

				// Determining the tags(if any) set on the error
				scope.SetTags(errors.Tags(err))

				// Capturing the error on Sentry
				eventId := sentry.CaptureException(err)
				log.Errorf(err, "Error captured in sentry with the event ID `%s`", *eventId)
			})
		} else {
			// Log the error sans sentry's event ID information
			log.Error(err)
		}

		if _panic {
			panic(err)
		}
	}
}

// Wrapper over sentry-go/http#HandleFunc
// Only calls the sentry handler if sentry was successfully initialized
func (wrapper *Sentry) HandleFunc(handler http.HandlerFunc) http.HandlerFunc {
	if wrapper.handler != nil {
		// If the sentry handler was initialized, call it's HandleFunc function
		return wrapper.handler.HandleFunc(handler)
	} else {
		// Simply return the handler in case the sentry handler was not initialized
		return handler
	}
}

// Wrapper over sentry-go/http#HandleFunc
// Only calls the sentry handler if sentry was successfully initialized
func (wrapper *Sentry) HandleHttpRouter(handler httprouter.Handle) httprouter.Handle {
	if wrapper.handler != nil {
		// If the sentry handler was initialized, call it's HandleFunc function
		return wrapper.handler.HandleHttpRouter(handler)
	} else {
		// Simply return the handler in case the sentry handler was not initialized
		return handler
	}
}

// SentryMiddleware use directly with mux
// returns http.Handler to directly use with router 
func (wrapper *Sentry) SentryMiddleware(next http.Handler) http.Handler {
    return wrapper.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
        next.ServeHTTP(w, r)
    })
}
