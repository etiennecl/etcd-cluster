package logger

import "github.com/clinia/x/logrusx"

type Provider interface {
	Logger() *logrusx.Logger
}
