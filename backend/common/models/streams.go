package models

import "context"

type Stream[T any] interface {
	Next(context.Context) bool
	Get() (*T, error)
}

type ClientMessaging interface {
	WriteJSON(any) error
}
