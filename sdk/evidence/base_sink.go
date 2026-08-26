package evidence

// BaseMemorySink is an in-memory append-only sink with sha256 hash-chain integrity.
type BaseMemorySink struct {
	AbstractSink
}

var _ Sink = (*BaseMemorySink)(nil)

func NewBaseMemorySink() *BaseMemorySink {
	return &BaseMemorySink{AbstractSink{Store: NewBaseMemoryStore(), Digester: BaseSHA256Digester{}}}
}
