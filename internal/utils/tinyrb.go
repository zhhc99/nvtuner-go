package tinyrb

type RingBuffer[T any] struct {
	data []T
	cap  int
}

func New[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data: make([]T, 0, capacity),
		cap:  capacity,
	}
}

func (rb *RingBuffer[T]) Push(v T) {
	if len(rb.data) >= rb.cap {
		rb.data = rb.data[1:]
	}
	rb.data = append(rb.data, v)
}

func (rb *RingBuffer[T]) Get() []T {
	return rb.data
}

func (rb *RingBuffer[T]) Clear() {
	rb.data = rb.data[:0]
}

func (rb *RingBuffer[T]) Len() int {
	return len(rb.data)
}
