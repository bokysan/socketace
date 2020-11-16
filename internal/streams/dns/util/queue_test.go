package util

import (
	errors "github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func Test_OutQueueChunking(t *testing.T) {
	o := &OutQueue{}

	require.Equal(t, uint16(0), o.NextSeqNo)
	require.Nil(t, o.NextChunk())

	data := []byte("012345678901234567890123456789012345678901234567890123456789")

	go func() {
		o.Write(data, 50)
	}()

	// Give the go subrutine a bit time to complete the write
	time.Sleep(200 * time.Millisecond)

	require.Len(t, o.out, 2)
	require.Equal(t, uint16(2), o.NextSeqNo)

	c1 := o.NextChunk()
	require.NotNil(t, c1)
	require.Equal(t, uint16(0), c1.SeqNo)

	c2 := o.NextChunk()
	require.NotNil(t, c2)
	require.Equal(t, uint16(0), c2.SeqNo)

	o.UpdateAcked(0)
	require.Len(t, o.acked, 1)

	c3 := o.NextChunk()
	require.NotNil(t, c3)
	require.Equal(t, uint16(1), c3.SeqNo)

	o.UpdateAcked(0)
	require.Len(t, o.acked, 1)

	c4 := o.NextChunk()
	require.NotNil(t, c4)
	require.Equal(t, uint16(1), c4.SeqNo)

	o.UpdateAcked(1)

	require.Nil(t, o.NextChunk())

}

func Test_OutQueue2(t *testing.T) {
	o := &OutQueue{}

	require.Equal(t, uint16(0), o.NextSeqNo)
	require.Nil(t, o.NextChunk())

	data := []byte("012345678901234567890123456789012345678901234567890123456789")

	go func() {
		o.Write(data, 50)
	}()

	// Give the go subrutine a bit time to complete the write
	time.Sleep(200 * time.Millisecond)

	o.UpdateAcked(5)

	c1 := o.NextChunk()
	require.NotNil(t, c1)
	require.Equal(t, uint16(0), c1.SeqNo)

	o.UpdateAcked(1)

	c2 := o.NextChunk()
	require.NotNil(t, c2)
	require.Equal(t, uint16(0), c2.SeqNo)

	o.UpdateAcked(0)

	c3 := o.NextChunk()
	require.Nil(t, c3)
}

func Test_OutQueue3(t *testing.T) {
	o := &OutQueue{}

	require.Equal(t, uint16(0), o.NextSeqNo)
	require.Nil(t, o.NextChunk())

	data := []byte("012345678901234567890123456789012345678901234567890123456789")

	go func() {
		o.Write(data, 80)
	}()

	// Give the go subrutine a bit time to complete the write
	time.Sleep(200 * time.Millisecond)

	require.Len(t, o.out, 1)
}
func Test_OutQueueBlocking(t *testing.T) {
	const MaxChunks = 16

	o := &OutQueue{}

	require.Equal(t, uint16(0), o.NextSeqNo)
	require.Nil(t, o.NextChunk())

	// Fill the queue first
	data := []byte("0123456789")
	long := ""

	for i := 0; i < MaxChunks; i++ {
		long += string(data)
	}

	go func() {
		_, err := o.Write([]byte(long), 10)
		require.NoError(t, err)
	}()

	// Give the go subrutine a bit time to complete the write
	time.Sleep(200 * time.Millisecond)

	require.Len(t, o.out, MaxChunks)

	// Next write should block
	wait1 := make(chan struct{}, 0)
	wait2 := make(chan struct{}, 0)
	go func() {
		wait1 <- struct{}{}
		o.Write([]byte("abcdefghij"), 20)
		wait2 <- struct{}{}
	}()

	// Make sure write is blocked
	select {
	case <-wait1:
	}

	var blocked bool
	select {
	case <-wait2:
		blocked = false
	case <-time.After(200 * time.Millisecond):
		blocked = true
	}

	require.True(t, blocked)

	require.Len(t, o.out, MaxChunks)

	for i := 0; i < MaxChunks; i++ {
		chunk := o.NextChunk()
		require.NotNil(t, chunk)
		require.Equal(t, uint16(i), chunk.SeqNo)
		o.UpdateAcked(uint16(i))
	}

	// Give the go subrutine a bit time to complete the write
	time.Sleep(200 * time.Millisecond)

	chunk := o.NextChunk()
	require.NotNil(t, chunk)

	require.Len(t, o.out, 1)
	o.UpdateAcked(o.out[0].SeqNo)

	var finished bool
	select {
	case <-wait2:
		finished = true
	case <-time.After(200 * time.Millisecond):
		finished = false
	}

	if !finished {
		log.Infof("out = %v", o.out)
	}

	require.True(t, finished)
	require.Len(t, o.out, 0)

}

func Test_InQueue1(t *testing.T) {
	i := &InQueue{}

	data1 := []byte("0123456789")

	i.Append(&Packet{
		SeqNo: 0,
		Data:  data1,
	})

	require.Equal(t, data1, i.in)
	require.Equal(t, uint16(1), i.NextSeqNo)
	require.Len(t, i.acked, 1)

	i.Append(&Packet{
		SeqNo: 0,
		Data:  data1,
	})

	require.Equal(t, data1, i.in)
	require.Equal(t, uint16(1), i.NextSeqNo)
	require.Len(t, i.acked, 1)

	i.Append(&Packet{
		SeqNo: 1,
		Data:  data1,
	})

	require.Equal(t, append(data1, data1...), i.in)
	require.Equal(t, uint16(2), i.NextSeqNo)
	require.Len(t, i.acked, 2)

	p := make([]byte, len(data1)*10)
	n, err := i.Read(p)

	require.NoError(t, err)
	require.Equal(t, len(data1)*2, n)
	p = p[0:n]
	require.Equal(t, append(data1, data1...), p)
	require.Equal(t, []byte{}, i.in)
}

func Test_InQueue2(t *testing.T) {
	i := &InQueue{}

	data1 := []byte("0123456789")

	for j := 0; j < MaxCachedChunks*2; j++ {
		i.Append(&Packet{
			SeqNo: uint16(j),
			Data:  data1,
		})
	}
	require.Equal(t, uint16(MaxCachedChunks*2), i.NextSeqNo)
	require.Len(t, i.acked, MaxCachedChunks)

}

func Test_InQueueBlocking(t *testing.T) {
	i := &InQueue{}

	data := []byte("0123456789")
	read := make([]byte, len(data)*2)

	require.Equal(t, uint16(0), i.NextSeqNo)

	// Next read should block
	wait1 := make(chan struct{}, 0)
	wait2 := make(chan error, 0)
	go func() {
		wait1 <- struct{}{}
		n, err := i.Read(read)

		if err != nil && n != len(data) {
			err = errors.Errorf("Expected len %d but got %d", len(data), n)
		}
		if err != nil && string(data) != string(read) {
			err = errors.Errorf("Expected %q but got %q", string(data), string(read))
		}

		wait2 <- err
	}()

	// Make sure write is blocked
	select {
	case <-wait1:
	}

	var blocked bool
	var readErr error
	select {
	case readErr = <-wait2:
		blocked = false
	case <-time.After(200 * time.Millisecond):
		blocked = true
	}

	require.True(t, blocked)
	require.NoError(t, readErr)

	require.NoError(t, i.Append(&Packet{
		SeqNo: 0,
		Data:  data,
	}))

	var finished bool
	select {
	case readErr = <-wait2:
		finished = true
	case <-time.After(200 * time.Millisecond):
		finished = false
	}

	require.True(t, finished)
	require.NoError(t, readErr)

	go func() {
		wait1 <- struct{}{}
		n, err := i.Read(read)

		if err != nil && n != len(data) {
			err = errors.Errorf("Expected len %d but got %d", len(data), n)
		}
		if err != nil && string(data) != string(read) {
			err = errors.Errorf("Expected %q but got %q", string(data), string(read))
		}

		wait2 <- err
	}()

	// Make sure write is blocked
	select {
	case <-wait1:
	}

	select {
	case readErr = <-wait2:
		blocked = false
	case <-time.After(200 * time.Millisecond):
		blocked = true
	}

	require.True(t, blocked)
	require.NoError(t, readErr)

}

// Test_InQueue3 will test receiving empty chunks
func Test_InQueueAddingEmtpyPacket(t *testing.T) {
	i := &InQueue{}
	err := i.Append(nil)
	require.NoError(t, err)
}

// Test_InQueue4 will test receiving out of order packets
func Test_InQueueOutOfOrder(t *testing.T) {
	var err error
	i := &InQueue{}

	err = i.Append(&Packet{
		SeqNo: 3,
		Data:  []byte("789"),
	})
	require.NoError(t, err)
	require.Len(t, i.in, 0)

	err = i.Append(&Packet{
		SeqNo: 1,
		Data:  []byte("123"),
	})
	require.NoError(t, err)
	require.Len(t, i.in, 0)

	err = i.Append(&Packet{
		SeqNo: 2,
		Data:  []byte("456"),
	})
	require.NoError(t, err)
	require.Len(t, i.in, 0)

	require.Len(t, i.future, 3)

	err = i.Append(&Packet{
		SeqNo: 0,
		Data:  []byte("0"),
	})
	require.NoError(t, err)
	require.Len(t, i.in, 10)

	require.Equal(t, i.in, []byte("0123456789"))

}

// Test_InQueue5 will test receiving out of sequence packets
func Test_InQueueInvalidSequence(t *testing.T) {
	i := &InQueue{}

	err := i.Append(&Packet{
		SeqNo: MaxCachedChunks + 500,
		Data:  []byte("789"),
	})
	require.Equal(t, err, ErrInvalidSequenceNumber)
	require.Len(t, i.in, 0)
}
