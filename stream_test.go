package goflow_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestAllMatch(t *testing.T) {
	ctx := t.Context()
	assert.True(t, stream.Of(ctx, 2, 4, 6).AllMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
	assert.False(t, stream.Of(ctx, 2, 3, 6).AllMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
}

func TestAllMatchEmpty(t *testing.T) {
	assert.True(t, stream.Empty[int]().AllMatch(func(_ context.Context, _ int) bool { return false }))
}

func ExampleStream_AllMatch() {
	ctx := context.Background()
	fmt.Println(stream.Of(ctx, 2, 4, 6).AllMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
	// Output: true
}

func TestAnyMatch(t *testing.T) {
	ctx := t.Context()
	assert.True(t, stream.Of(ctx, 1, 2, 3).AnyMatch(func(_ context.Context, n int) bool { return n == 2 }))
	assert.False(t, stream.Of(ctx, 1, 3, 5).AnyMatch(func(_ context.Context, n int) bool { return n == 2 }))
}

func TestAnyMatchEmpty(t *testing.T) {
	assert.False(t, stream.Empty[int]().AnyMatch(func(_ context.Context, _ int) bool { return true }))
}

func ExampleStream_AnyMatch() {
	ctx := context.Background()
	fmt.Println(stream.Of(ctx, 1, 2, 3).AnyMatch(func(_ context.Context, n int) bool { return n > 2 }))
	// Output: true
}

func TestCollect(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, stream.Of(t.Context(), "a", "b").Collect())
	assert.Nil(t, stream.Empty[int]().Collect())
}

func ExampleStream_Collect() {
	ctx := context.Background()
	fmt.Println(stream.Of(ctx, "a", "b", "c").Collect())
	// Output: [a b c]
}

func TestDistinct(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, "a", "b", "a", "c", "b").Distinct(func(s string) string {
		return s
	}).Collect()
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func ExampleStream_Distinct() {
	ctx := context.Background()
	got := stream.Of(ctx, "a", "b", "a", "c", "b").Distinct(func(s string) string {
		return s
	}).Collect()
	fmt.Println(got)
	// Output: [a b c]
}

func TestFanOut(t *testing.T) {
	ctx := t.Context()
	parts := stream.Of(ctx, 1, 2, 3, 4, 5).FanOut(2)
	assert.Len(t, parts, 2)

	// consume both partitions concurrently to avoid deadlock
	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = parts[1].Collect()

		close(done)
	}()

	results[0] = parts[0].Collect()

	<-done

	assert.Equal(t, []int{1, 3, 5}, results[0])
	assert.Equal(t, []int{2, 4}, results[1])
}

func TestFanOutCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	parts := stream.Of(ctx, 1, 2, 3).FanOut(2)
	assert.Nil(t, parts)
}

func ExampleStream_FanOut() {
	ctx := context.Background()
	parts := stream.Of(ctx, 1, 2, 3, 4, 5).FanOut(2)

	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = parts[1].Collect()

		close(done)
	}()

	results[0] = parts[0].Collect()

	<-done

	fmt.Println(results[0])
	fmt.Println(results[1])
	// Output:
	// [1 3 5]
	// [2 4]
}

func TestFilter(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3, 4, 5, 6).Filter(func(_ context.Context, n int) bool {
		return n%2 == 0
	}).Collect()
	assert.Equal(t, []int{2, 4, 6}, got)
}

func TestFilterEmpty(t *testing.T) {
	got := stream.Empty[int]().Filter(func(_ context.Context, _ int) bool {
		return true
	}).Collect()
	assert.Nil(t, got)
}

func ExampleStream_Filter() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3, 4, 5, 6).Filter(func(_ context.Context, n int) bool {
		return n%2 == 0
	}).Collect()
	fmt.Println(got)
	// Output: [2 4 6]
}

func TestFindFirst(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 10, 20, 30).FindFirst()
	assert.True(t, ok)
	assert.Equal(t, 10, v)
}

func TestFindFirstEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().FindFirst()
	assert.False(t, ok)
	assert.Zero(t, v)
}

func ExampleStream_FindFirst() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 10, 20, 30).FindFirst()
	fmt.Println(v, ok)
	// Output: 10 true
}

func TestFindFirstMatch(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 1, 2, 3, 4).FindFirstMatch(func(_ context.Context, n int) bool { return n > 2 })
	assert.True(t, ok)
	assert.Equal(t, 3, v)
}

func TestFindFirstMatchNone(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 1, 2, 3).FindFirstMatch(func(_ context.Context, n int) bool { return n > 10 })
	assert.False(t, ok)
	assert.Zero(t, v)
}

func TestFindFirstMatchEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().FindFirstMatch(func(_ context.Context, _ int) bool { return true })
	assert.False(t, ok)
	assert.Zero(t, v)
}

func ExampleStream_FindFirstMatch() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 1, 2, 3, 4).FindFirstMatch(func(_ context.Context, n int) bool { return n > 2 })
	fmt.Println(v, ok)
	// Output: 3 true
}

func TestForEach(t *testing.T) {
	ctx := t.Context()

	var got []int

	err := stream.Of(ctx, 1, 2, 3).ForEach(func(_ context.Context, v int) error {
		got = append(got, v)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestForEachError(t *testing.T) {
	ctx := t.Context()
	errBoom := errors.New("boom")

	var count int

	err := stream.Of(ctx, 1, 2, 3, 4, 5).ForEach(func(_ context.Context, _ int) error {
		count++
		if count == 3 {
			return errBoom
		}

		return nil
	})
	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, 3, count)
}

func TestForEachEmpty(t *testing.T) {
	err := stream.Empty[int]().ForEach(func(_ context.Context, _ int) error {
		t.Fatal("should not be called")
		return nil
	})
	require.NoError(t, err)
}

func ExampleStream_ForEach() {
	ctx := context.Background()
	_ = stream.Of(ctx, 1, 2, 3).ForEach(func(_ context.Context, v int) error {
		fmt.Println(v)
		return nil
	})
	// Output:
	// 1
	// 2
	// 3
}

func TestIter(t *testing.T) {
	ctx := t.Context()

	var got []int
	for v := range stream.Of(ctx, 1, 2, 3).Iter() {
		got = append(got, v)
	}

	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestIterEmpty(t *testing.T) {
	var got []int
	for v := range stream.Empty[int]().Iter() {
		got = append(got, v)
	}

	assert.Nil(t, got)
}

func TestIterBreak(t *testing.T) {
	ctx := t.Context()

	var got []int

	for v := range stream.Of(ctx, 1, 2, 3, 4, 5).Iter() {
		if v > 2 {
			break
		}

		got = append(got, v)
	}

	assert.Equal(t, []int{1, 2}, got)
}

func ExampleStream_Iter() {
	ctx := context.Background()
	for v := range stream.Of(ctx, 1, 2, 3).Iter() {
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
}

func TestIter2(t *testing.T) {
	ctx := t.Context()

	var (
		keys []int
		vals []string
	)

	for i, v := range stream.Of(ctx, "a", "b", "c").Iter2() {
		keys = append(keys, i)
		vals = append(vals, v)
	}

	assert.Equal(t, []int{0, 1, 2}, keys)
	assert.Equal(t, []string{"a", "b", "c"}, vals)
}

func TestMax(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Max(cmp.Compare[int])
	assert.True(t, ok)
	assert.Equal(t, 5, v)
}

func TestMaxEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().Max(cmp.Compare[int])
	assert.False(t, ok)
	assert.Zero(t, v)
}

func ExampleStream_Max() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Max(cmp.Compare[int])
	fmt.Println(v, ok)
	// Output: 5 true
}

func TestMin(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Min(cmp.Compare[int])
	assert.True(t, ok)
	assert.Equal(t, 1, v)
}

func TestMinEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().Min(cmp.Compare[int])
	assert.False(t, ok)
	assert.Zero(t, v)
}

func ExampleStream_Min() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Min(cmp.Compare[int])
	fmt.Println(v, ok)
	// Output: 1 true
}

func TestNoneMatch(t *testing.T) {
	ctx := t.Context()
	assert.True(t, stream.Of(ctx, 1, 3, 5).NoneMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
	assert.False(t, stream.Of(ctx, 1, 2, 5).NoneMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
}

func TestNoneMatchEmpty(t *testing.T) {
	assert.True(t, stream.Empty[int]().NoneMatch(func(_ context.Context, _ int) bool { return true }))
}

func TestPeek(t *testing.T) {
	ctx := t.Context()

	var peeked []int

	got := stream.Of(ctx, 1, 2, 3).Peek(func(_ context.Context, n int) {
		peeked = append(peeked, n)
	}).Collect()
	assert.Equal(t, []int{1, 2, 3}, got)
	assert.Equal(t, []int{1, 2, 3}, peeked)
}

func ExampleStream_Peek() {
	ctx := context.Background()

	var peeked []int

	got := stream.Of(ctx, 1, 2, 3).Peek(func(_ context.Context, n int) {
		peeked = append(peeked, n)
	}).Collect()
	fmt.Println(got)
	fmt.Println(peeked)
	// Output:
	// [1 2 3]
	// [1 2 3]
}

func TestProcess(t *testing.T) {
	ctx := t.Context()

	var (
		mu  sync.Mutex
		got []int
	)

	err := stream.Of(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Process(3, func(_ context.Context, v int) error {
		mu.Lock()

		got = append(got, v)
		mu.Unlock()

		return nil
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, got)
}

func TestProcessError(t *testing.T) {
	ctx := t.Context()
	errBoom := errors.New("boom")
	err := stream.Of(ctx, 1, 2, 3).Process(2, func(_ context.Context, v int) error {
		if v == 2 {
			return errBoom
		}

		return nil
	})
	require.ErrorIs(t, err, errBoom)
}

func TestProcessLimit(t *testing.T) {
	ctx := t.Context()

	var (
		active atomic.Int32
		peak   atomic.Int32
	)

	err := stream.Of(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Process(3, func(_ context.Context, _ int) error {
		cur := active.Add(1)

		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
		active.Add(-1)

		return nil
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, peak.Load(), int32(3))
}

func TestProcessCancelSingleError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	ch := make(chan int)
	close(ch)

	err := stream.From(ctx, ch).Process(4, func(context.Context, int) error { return nil })
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, context.Canceled.Error(), err.Error(), "expected one context error, not one per worker")
}

// WithGroupOptions reaches the worker group: a limit of 1 runs one worker at a time.
func TestProcessWithGroupOptions(t *testing.T) {
	var running, peak atomic.Int64

	err := stream.Of(t.Context(), 1, 2, 3, 4, 5, 6).Process(4, func(context.Context, int) error {
		cur := running.Add(1)
		defer running.Add(-1)

		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(5 * time.Millisecond)

		return nil
	}, stream.WithGroupOptions(gofuncy.WithLimit(1)))
	require.NoError(t, err)
	assert.Equal(t, int64(1), peak.Load())
}

func TestReverse(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3).Reverse().Collect()
	assert.Equal(t, []int{3, 2, 1}, got)
}

func TestReverseEmpty(t *testing.T) {
	got := stream.Empty[int]().Reverse().Collect()
	assert.Nil(t, got)
}

func ExampleStream_Reverse() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3).Reverse().Collect()
	fmt.Println(got)
	// Output: [3 2 1]
}

func TestSkip(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Skip(2).Collect()
	assert.Equal(t, []int{3, 4, 5}, got)
}

func TestSkipAll(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3).Skip(10).Collect()
	assert.Nil(t, got)
}

func ExampleStream_Skip() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Skip(2).Collect()
	fmt.Println(got)
	// Output: [3 4 5]
}

func TestSort(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 3, 1, 4, 1, 5).Sort(cmp.Compare[int]).Collect()
	assert.Equal(t, []int{1, 1, 3, 4, 5}, got)
}

func ExampleStream_Sort() {
	ctx := context.Background()
	got := stream.Of(ctx, 3, 1, 4, 1, 5).Sort(cmp.Compare[int]).Collect()
	fmt.Println(got)
	// Output: [1 1 3 4 5]
}

func TestTake(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Take(3).Collect()
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestTakeMoreThanSource(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2).Take(10).Collect()
	assert.Equal(t, []int{1, 2}, got)
}

func ExampleStream_Take() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Take(3).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
}

func TestTee(t *testing.T) {
	ctx := t.Context()
	streams := stream.Of(ctx, 1, 2, 3).Tee(2)
	assert.Len(t, streams, 2)

	// consume both concurrently to avoid deadlock
	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = streams[1].Collect()

		close(done)
	}()

	results[0] = streams[0].Collect()

	<-done

	assert.Equal(t, []int{1, 2, 3}, results[0])
	assert.Equal(t, []int{1, 2, 3}, results[1])
}

func ExampleStream_Tee() {
	ctx := context.Background()
	streams := stream.Of(ctx, 1, 2, 3).Tee(2)

	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = streams[1].Collect()

		close(done)
	}()

	results[0] = streams[0].Collect()

	<-done

	fmt.Println(results[0])
	fmt.Println(results[1])
	// Output:
	// [1 2 3]
	// [1 2 3]
}

func TestThrottle(t *testing.T) {
	ctx := t.Context()
	start := time.Now()
	got := stream.Of(ctx, 1, 2, 3).Throttle(20 * time.Millisecond).Collect()
	elapsed := time.Since(start)

	assert.Equal(t, []int{1, 2, 3}, got)
	// 3 items with 20ms throttle: first immediate, then 2 waits = ~40ms minimum
	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond)
}

func ExampleStream_Throttle() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3).Throttle(1 * time.Millisecond).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
}
