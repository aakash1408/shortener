package apperr

import(
	"errors"
	"log/slog"
	"net/http"
)

var (
	ErrNotFound = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden = errors.New("forbidden")
	ErrConflict = errors.New("conflict")
	ErrExpired = errors.New("url expired")
)

type errWithAttrs struct{
	error
	attrs []slog.Attr
}

func (e *errWithAttrs) Unwrap() error{
	return e.error
}

func (e *errWithAttrs) Attrs() []slog.Attr {
	return e.attrs
}

func WithAttrs(err error, args ...any) error{
	return &errWithAttrs{
		error: err,
		attrs: argsToAttrs(args),
	}
}

func argsToAttrs(args []any) []slog.Attr{
	var attrs []slog.Attr
	for i := 0; i+1 < len(args); i +=2{
		key, ok := args[i].(string)
		if !ok{
			continue
		}
		attrs = append(attrs, slog.Any(key, args[i+1]))
	}
	return attrs
}

func StatusCode(err error) int{
	switch{
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrExpired):
		return http.StatusGone
	default:
		return http.StatusInternalServerError
	}
}