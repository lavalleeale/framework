package framework

type QueueNotFoundError struct{}

func (e QueueNotFoundError) Error() string {
	return "queue not found"
}

type JobNotMarshalableError struct{}

func (e JobNotMarshalableError) Error() string {
	return "job not marshalable"
}
