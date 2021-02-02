package broadcast

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TypedString string

func TestBroadcast(t *testing.T) {
	assert := assert.New(t)
	b := NewTypedBroadcaster()
	type Args struct {
		Name string
	}

	ch1 := make(chan string, 1)
	ch2 := make(chan string, 1)
	b.AddListener(func(ctx context.Context, args *Args) {
		assert.Equal("VALUE", ctx.Value("KEY"))
		ch1 <- args.Name
	})
	b.AddListener(func(ctx context.Context, args *Args) {
		assert.Equal("VALUE", ctx.Value("KEY"))
		ch2 <- args.Name
	})
	ctx := context.Background()
	ctx = context.WithValue(ctx, "KEY", "VALUE")
	b.Notify(ctx, &Args{Name: "Hello"})
	v := <-ch1
	assert.Equal("Hello", v)
	v = <-ch2
	assert.Equal("Hello", v)

	var count int32
	b.AddListener(func(ctx context.Context, s TypedString) {
		atomic.AddInt32(&count, 1)
	})
	b.AddListener(func(ctx context.Context, s string) {
		atomic.AddInt32(&count, 2)
	})
	b.Notify(context.Background(), TypedString("A"))
	time.Sleep(1 * time.Millisecond)
	assert.Equal(int32(1), count)

}

func TestBroadcastErrTorlerate(t *testing.T) {
	logStack = func(tag string, r interface{}) {}
	assert := assert.New(t)
	b := NewTypedBroadcaster()

	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	b.AddListener(func(ctx context.Context, args int) {
		if args == 0 {
			panic("error")
		}
		ch1 <- args
	})
	b.AddListener(func(ctx context.Context, args int) {
		if args == 0 {
			panic("error")
		}
		ch2 <- args
	})
	ctx := context.Background()
	b.Notify(ctx, 0)
	b.Notify(ctx, 1)
	v := <-ch1
	assert.Equal(1, v)
	v = <-ch2
	assert.Equal(1, v)

}

func TestBroadcastStop(t *testing.T) {
	b := NewTypedBroadcaster()
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	l1 := b.AddListener(func(ctx context.Context, args int) {
		ch1 <- args
	})
	l2 := b.AddListener(func(ctx context.Context, args int) {
		ch2 <- args
	})
	ctx := context.Background()
	b.Stop()
	l1.Wait()
	l2.Wait()
	b.Notify(ctx, 1)
	select {
	case <-ch1:
		t.Fatal("should not recv anything")
	case <-ch2:
		t.Fatal("should not recv anything")
	default:
	}
}
