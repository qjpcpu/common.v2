package joint

import (
	"sync"
	"testing"
	"time"
)

func TestSimplePipe(t *testing.T) {
	memo := make(map[int]int)
	l := new(sync.RWMutex)

	in, out := make(chan int), make(chan int)
	pipe, err := Pipe(in, out)
	if err != nil {
		t.Fatalf("create pipe %v", err)
	}
	defer pipe.Breakoff()
	wg := new(sync.WaitGroup)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(base int) {
			for j := 0; j < 100; j++ {
				num := base*1000 + j
				l.Lock()
				memo[num]++
				l.Unlock()
				in <- num
			}
			wg.Done()
		}(i)
	}
	for i := 0; i < 3; i++ {
		go func() {
			for {
				num := <-out
				l.Lock()
				memo[num]++
				l.Unlock()
			}
		}()
	}
	wg.Wait()
	// stop send
	close(in)
	<-pipe.DoneC()
	// wait write consume the last data
	time.Sleep(1 * time.Millisecond)
	for num, v := range memo {
		if v != 2 {
			t.Fatal("lost data", num)
		}
	}
}

func TestHaltNow(t *testing.T) {
	in, out := make(chan int), make(chan int)
	pipe, err := Pipe(in, out)
	if err != nil {
		t.Fatalf("create pipe %v", err)
	}
	for i := 0; i < 50; i++ {
		go func(base int) {
			for j := 0; j < 100; j++ {
				num := base*1000 + j
				in <- num
			}
		}(i)
	}
	for i := 0; i < 3; i++ {
		go func() {
			for {
				time.Sleep(time.Second)
				<-out
			}
		}()
	}
	pipe.Breakoff()
	select {
	case <-pipe.DoneC():
	case <-time.After(5 * time.Second):
		t.Fatal("should halt right now")
	}

}

func TestSeq(t *testing.T) {
	size := 1000
	in, out := make(chan int, size), make(chan int)
	pipe, err := Pipe(in, out)
	if err != nil {
		t.Fatalf("create pipe %v", err)
	}
	for i := 0; i < size; i++ {
		in <- i
	}
	defer pipe.Breakoff()
	for i := 0; i < size; i++ {
		num := <-out
		if num != i {
			t.Fatal("bad sequence")
		}
	}

}

func TestBlocked(t *testing.T) {
	in, out := make(chan int), make(chan int)
	pipe, err := Pipe(in, out)
	if err != nil {
		t.Fatalf("create pipe %v", err)
	}
	defer pipe.Breakoff()
	cap := 5
	if err = pipe.SetCap(uint64(cap)); err != nil {
		t.Fatalf("set cap fail %v", err)
	}
	for i := 0; i < cap; i++ {
		in <- i
	}
	select {
	case in <- 100:
		t.Fatal("should blocked")
	case <-time.After(time.Millisecond):
	}
	in, out = make(chan int), make(chan int)
	pipe, _ = Pipe(in, out)
	pipe.SetCap(1)
	in <- 1
	select {
	case in <- 100:
		t.Fatal("should blocked")
	case <-time.After(time.Millisecond):
	}
}

func TestDynamicCap(t *testing.T) {
	in, out := make(chan int), make(chan int)
	pipe, err := Pipe(in, out)
	if err != nil {
		t.Fatalf("create pipe %v", err)
	}
	defer pipe.Breakoff()
	cap := 5
	if err = pipe.SetCap(uint64(cap)); err != nil {
		t.Fatalf("set cap fail %v", err)
	}
	for i := 0; i < cap; i++ {
		in <- i
	}
	select {
	case in <- 100:
		t.Fatal("should blocked")
	case <-time.After(time.Millisecond):
	}
	// drain out
	for i := 0; i < cap; i++ {
		<-out
	}
	cap = 3
	if err = pipe.SetCap(uint64(cap)); err != nil {
		t.Fatalf("set cap fail %v", err)
	}
	for i := 0; i < cap; i++ {
		in <- i
	}
	select {
	case in <- 100:
		t.Fatal("should blocked")
	case <-time.After(time.Millisecond):
	}
	bigCap := 20
	if err = pipe.SetCap(uint64(bigCap)); err != nil {
		t.Fatalf("set cap fail %v", err)
	}
	for i := 0; i < bigCap-cap; i++ {
		select {
		case in <- i:
		case <-time.After(time.Millisecond):
			t.Fatal("should not blocked")
		}
	}
	select {
	case in <- 100:
		t.Fatal("should blocked")
	case <-time.After(time.Millisecond):
	}
}

func TestFilter(t *testing.T) {
	in, out := make(chan int), make(chan int)
	pipe, err := Pipe(in, out)
	if err != nil {
		t.Fatalf("create pipe %v", err)
	}
	defer pipe.Breakoff()
	pipe.SetFilter(func(v interface{}) bool {
		num := v.(int)
		return num >= 100
	})
	data := []int{1, 2, 100, 200, 300, 50, 20, 500, 20, 101, 200}
	validData := make(map[int]int)
	var total int
	for _, v := range data {
		if v >= 100 {
			validData[v]++
			total++
		}
		in <- v
	}
	for i := 0; i < total; i++ {
		outv := <-out
		t.Logf("TestFilter get %d", outv)
		if _, ok := validData[outv]; !ok {
			t.Fatalf("should not get %v", outv)
		}
		validData[outv]--
	}
	select {
	case <-out:
		t.Fatal("should blocked")
	case <-time.After(time.Millisecond):
	}
	if pipe.Len() != 0 {
		t.Fatal("should be empty")
	}
	for k, cnt := range validData {
		if cnt != 0 {
			t.Fatalf("lost %d", k)
		}
	}
}
