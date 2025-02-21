package controller

type ChannelMessage[T any] struct {
	Error error
	Value T
}
