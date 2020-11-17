package util

import (
	"github.com/pkg/errors"
	"sync"
	"time"
)

// Keep at most 128 acked chunks from the other party
const MaxCachedChunks = 128

var (
	ErrInvalidSequenceNumber = errors.New("invalid chunk sequence")
	ErrStreamBroken          = errors.New("stream broken")
	ErrDeadlineExceeded      = errors.New("deadline exceeded")
)

// Packet represents a part of data exchanged between the client and the server
type Packet struct {
	SeqNo uint16 // The sequence number of the chunk.
	Data  []byte // The chunk data
}

type InQueue struct {
	NextSeqNo      uint16     // The id of the next Packet to be written to the queue
	mutex          sync.Mutex // Synchronization mutex for accessing the queue
	in             []byte     // The input buffer
	future         []*Packet  // Chunks which were received out of order and are waiting to be written to the queue
	acked          []uint16   // List of the last few acked Packets
	queueMutex     sync.Mutex // Synchronization mutex for accessing the queue
	queueHasData   bool       // Boolean specifiying if there's any data in the queue
	queueNotifiers []func()   // A list of waiters to notify when the queue has data
	readDeadline   time.Time
}

// HasData returns true if there's any data waiting in the queue to be read
func (q *InQueue) HasData() bool {
	return q.queueHasData
}

func (q *InQueue) checkQueueHasAny() {
	var hasAny bool

	q.queueMutex.Lock()
	hasAny = len(q.in) > 0
	q.queueHasData = hasAny
	q.queueMutex.Unlock()

	if hasAny {
		// Notify waiting notifier
		q.queueMutex.Lock()
		for _, f := range q.queueNotifiers {
			f()
		}
		q.queueNotifiers = q.queueNotifiers[0:0]
		q.queueMutex.Unlock()
	}
}

func (q *InQueue) waitNonEmtpyQueue() error {
	if !q.readDeadline.IsZero() {
		if q.readDeadline.Before(time.Now()) {
			return ErrDeadlineExceeded
		}
	}
	q.queueMutex.Lock()

	// If queue is not empty, return straight away
	if q.queueHasData {
		q.queueMutex.Unlock()
		return nil
	}

	wait := make(chan struct{}, 0)
	q.queueNotifiers = append(q.queueNotifiers, func() {
		wait <- struct{}{}
	})
	q.queueMutex.Unlock()

	if q.readDeadline.IsZero() {
		// Wait for the notification that the queue has been filled
		select {
		case <-wait:
		}
		return nil
	} else {
		// Wait for the notification that the queue has been filled
		select {
		case <-time.After(q.readDeadline.Sub(time.Now())):
			return ErrDeadlineExceeded
		case <-wait:
			return nil
		}
	}
}

func (q *InQueue) Read(p []byte) (n int, err error) {
	// Block until data is available
	err = q.waitNonEmtpyQueue()
	if err != nil {
		return
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	copied := copy(p, q.in)
	q.in = q.in[copied:]

	q.checkQueueHasAny()
	return copied, nil
}

// Try to append the Packet to our byte list. Returns an error if out of order
func (q *InQueue) Append(val *Packet) error {
	if val == nil {
		return nil
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.isAcked(val.SeqNo) {
		// Ignore already acked Packet
		return nil
	}

	if val.SeqNo == q.NextSeqNo {
		// Good, the Packet we expected
		q.appendPacket(val)
		q.acked = append(q.acked, val.SeqNo)

		// Now add any Packet that we already received out-of-sequence
		added := true
		for len(q.future) > 0 && added {
			// See if any future Packet need to be added
			added = false
			for i, f := range q.future {
				if f.SeqNo == q.NextSeqNo {
					q.appendPacket(f)
					q.future = append(q.future[0:i], q.future[i+1:]...)
					added = true
					break
				}
			}
		}

		// Shorten the list of acked chunks
		if len(q.acked) > MaxCachedChunks {
			// Remove first acked chunk
			q.acked = q.acked[1:]
		}
	} else {
		// Check if this is not a "weird" chunk
		inWindow := false
		for i := q.NextSeqNo + 1; i != q.NextSeqNo+MaxCachedChunks; i++ {
			if i == val.SeqNo {
				inWindow = true
			}
		}
		if !inWindow {
			return errors.Wrapf(ErrInvalidSequenceNumber, "Received #%d but expected #%d. Acked: %v", val.SeqNo, q.NextSeqNo, q.acked)
		}

		// Out of order chunk, store it for later
		q.future = append(q.future, val)
		q.acked = append(q.acked, val.SeqNo)
	}

	q.checkQueueHasAny()
	return nil

}

func (q *InQueue) isAcked(val uint16) bool {
	for _, r := range q.acked {
		if r == val {
			return true
		}
	}
	return false
}

func (q *InQueue) appendPacket(val *Packet) {
	// Write directy to the In
	q.in = append(q.in, val.Data...)
	q.NextSeqNo += 1
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (q *InQueue) SetReadDeadline(t time.Time) error {
	q.readDeadline = t
	return nil
}

type OutQueue struct {
	NextSeqNo      uint16       // Next chunk to be put into queue will get this ID
	OnChunkAdded   func() error // Callback function when a new chunk is added
	mutex          sync.Mutex   // Synchronization mutex for accessing the queue
	out            []*Packet    // The outqueue
	acked          []uint16     // Sliding window of acked packages
	queueMutex     sync.Mutex   // Synchronization mutex for accessing the queue
	queueHasData   bool         // Boolean specifiying if the queue is full or not
	queueNotifiers []func()     // A list of waiters to notify when the queue is emptied
	writeDeadline  time.Time
}

// NextChunk will return the first non-acked chunk from the queue. It will return nil if the queue is empty
func (q *OutQueue) NextChunk() *Packet {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.cleanAckedChunks()

	if len(q.out) == 0 {
		return nil
	} else {
		return q.out[0]
	}
}

// UpdateAcked will add the acked sequence number to the list and remove the chunk from the outboud queue.
func (q *OutQueue) UpdateAcked(seqNo uint16) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	for _, a := range q.acked {
		if a == seqNo {
			// Already on the list, nothing to do
			return
		}
	}

	q.acked = append(q.acked, seqNo)
	q.cleanAckedChunks()
}

// cleanAckedChunks will remove all acked chunks from the queue
func (q *OutQueue) cleanAckedChunks() {
	for _, a := range q.acked {
		for p, c := range q.out {
			if c.SeqNo == a {
				q.out = append(q.out[0:p], q.out[p+1:]...)
				break
			}
		}
	}
	if len(q.acked) > MaxCachedChunks {
		q.acked = q.acked[0:MaxCachedChunks]
	}

	q.checkQueueFull()
}

func (q *OutQueue) checkQueueFull() {
	var full bool

	q.queueMutex.Lock()
	full = len(q.out) > 0
	q.queueHasData = full
	q.queueMutex.Unlock()

	if !full {
		// Notify waiting notifier
		q.queueMutex.Lock()
		for _, f := range q.queueNotifiers {
			f()
		}
		q.queueNotifiers = q.queueNotifiers[0:0]
		q.queueMutex.Unlock()
	}
}

func (q *OutQueue) waitEmptyQueue() error {
	if !q.writeDeadline.IsZero() {
		if q.writeDeadline.Before(time.Now()) {
			return ErrDeadlineExceeded
		}
	}
	q.queueMutex.Lock()

	// If queue is not full, return straight away
	if !q.queueHasData {
		q.queueMutex.Unlock()
		return nil
	}

	wait := make(chan struct{}, 0)
	q.queueNotifiers = append(q.queueNotifiers, func() {
		wait <- struct{}{}
	})
	q.queueMutex.Unlock()

	if q.writeDeadline.IsZero() {
		// Wait for the notification that the queue has emptied
		select {
		case <-wait:
		}
		return nil
	} else {
		// Wait for the notification that the queue has emptied
		select {
		case <-time.After(q.writeDeadline.Sub(time.Now())):
			return ErrDeadlineExceeded
		case <-wait:
			return nil
		}
	}
}

func (q *OutQueue) addChunk(data []byte) error {

	q.mutex.Lock()
	q.out = append(q.out, &Packet{
		SeqNo: q.NextSeqNo,
		Data:  data,
	})

	q.NextSeqNo += 1
	q.mutex.Unlock()

	if q.OnChunkAdded != nil {
		return q.OnChunkAdded()
	}

	return nil
}

// Write will create packets out of the given byte stream. Make sure that the writes are as large as possible,
// otherwise Packet will get quite small.
func (q *OutQueue) Write(b []byte, mtu uint32) (n int, err error) {
	err = q.waitEmptyQueue()
	if err != nil {
		return
	}

	for len(b) > 0 {
		var data []byte
		if uint32(len(b)) > mtu {
			data = b[0:mtu]
			b = b[mtu:]
		} else {
			data = b
			b = b[0:0]
		}
		err = q.addChunk(data)
		n += len(data)
		if err != nil {
			return
		}
	}

	q.checkQueueFull()
	return n, q.waitEmptyQueue()
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (q *OutQueue) SetWriteDeadline(t time.Time) error {
	q.writeDeadline = t
	return nil
}
