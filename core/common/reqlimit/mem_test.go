package reqlimit_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/labring/aiproxy/core/common/reqlimit"
)

func TestNewInMemoryRateLimiter(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()
	if rl == nil {
		t.Fatal("NewInMemoryRateLimiter should return a non-nil instance")
	}
}

func TestPushRequestBasic(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	normalCount, overCount, secondCount := rl.PushRequest("group1", "model1", 10, 60*time.Second, 1)

	if normalCount != 1 {
		t.Errorf("Expected normalCount to be 1, got %d", normalCount)
	}
	if overCount != 0 {
		t.Errorf("Expected overCount to be 0, got %d", overCount)
	}
	if secondCount != 1 {
		t.Errorf("Expected secondCount to be 1, got %d", secondCount)
	}
}

func TestPushRequestRateLimit(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	maxReq := int64(2)
	duration := 60 * time.Second

	for i := 0; i < 4; i++ {
		normalCount, overCount, _ := rl.PushRequest("group1", "model1", maxReq, duration, 1)

		if i < 2 {
			if normalCount != int64(i+1) {
				t.Errorf("Request %d: expected normalCount %d, got %d", i+1, i+1, normalCount)
			}
			if overCount != 0 {
				t.Errorf("Request %d: expected overCount 0, got %d", i+1, overCount)
			}
		} else if i == 2 {
			if normalCount != 3 {
				t.Errorf("Request %d: expected normalCount 3, got %d", i+1, normalCount)
			}
			if overCount != 0 {
				t.Errorf("Request %d: expected overCount 0, got %d", i+1, overCount)
			}
		} else {
			if normalCount != 3 {
				t.Errorf("Request %d: expected normalCount 3, got %d", i+1, normalCount)
			}
			if overCount != 1 {
				t.Errorf("Request %d: expected overCount 1, got %d", i+1, overCount)
			}
		}
	}
}

func TestPushRequestUnlimited(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	for i := 0; i < 5; i++ {
		normalCount, overCount, _ := rl.PushRequest("group1", "model1", 0, 60*time.Second, 1)

		if normalCount != int64(i+1) {
			t.Errorf("Request %d: expected normalCount %d, got %d", i+1, i+1, normalCount)
		}
		if overCount != 0 {
			t.Errorf("Request %d: expected overCount 0, got %d", i+1, overCount)
		}
	}
}

func TestGetRequest(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	rl.PushRequest("group1", "model1", 10, 60*time.Second, 1)
	rl.PushRequest("group1", "model2", 10, 60*time.Second, 1)
	rl.PushRequest("group2", "model1", 10, 60*time.Second, 1)

	totalCount, secondCount := rl.GetRequest("group1", "model1", 60*time.Second)
	if totalCount != 1 {
		t.Errorf("Expected totalCount 1, got %d", totalCount)
	}
	if secondCount != 1 {
		t.Errorf("Expected secondCount 1, got %d", secondCount)
	}

	totalCount, _ = rl.GetRequest("*", "*", 60*time.Second)
	if totalCount != 3 {
		t.Errorf("Expected totalCount 3 for wildcard query, got %d", totalCount)
	}

	totalCount, _ = rl.GetRequest("group1", "*", 60*time.Second)
	if totalCount != 2 {
		t.Errorf("Expected totalCount 2 for group1 wildcard, got %d", totalCount)
	}
}

func TestMultipleGroupsAndModels(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	groups := []string{"group1", "group2", "group3"}
	models := []string{"model1", "model2"}

	for _, group := range groups {
		for _, model := range models {
			rl.PushRequest(group, model, 10, 60*time.Second, 1)
		}
	}

	totalCount, _ := rl.GetRequest("*", "*", 60*time.Second)
	expected := len(groups) * len(models)
	if totalCount != int64(expected) {
		t.Errorf("Expected totalCount %d, got %d", expected, totalCount)
	}
}

func TestTimeWindowCleanup(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	rl.PushRequest("group1", "model1", 10, 2*time.Second, 1)

	totalCount, _ := rl.GetRequest("group1", "model1", 2*time.Second)
	if totalCount != 1 {
		t.Errorf("Expected totalCount 1, got %d", totalCount)
	}

	time.Sleep(3 * time.Second)

	totalCount, _ = rl.GetRequest("group1", "model1", 2*time.Second)
	if totalCount != 0 {
		t.Errorf("Expected totalCount 0 after cleanup, got %d", totalCount)
	}
}

func TestConcurrentAccess(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	const numGoroutines = 100
	const requestsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				rl.PushRequest("group1", "model1", 0, 60*time.Second, 1)
			}
		}(i)
	}

	wg.Wait()

	totalCount, _ := rl.GetRequest("group1", "model1", 60*time.Second)
	expected := int64(numGoroutines * requestsPerGoroutine)
	if totalCount != expected {
		t.Errorf("Expected totalCount %d, got %d", expected, totalCount)
	}
}

func TestConcurrentDifferentKeys(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			group := fmt.Sprintf("group%d", id%5)
			model := fmt.Sprintf("model%d", id%3)
			rl.PushRequest(group, model, 10, 60*time.Second, 1)
		}(i)
	}

	wg.Wait()

	// 验证总数
	totalCount, _ := rl.GetRequest("*", "*", 60*time.Second)
	if totalCount != int64(numGoroutines) {
		t.Errorf("Expected totalCount %d, got %d", numGoroutines, totalCount)
	}
}

func TestRateLimitWithOverflow(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	maxReq := 5
	duration := 60 * time.Second

	for i := 0; i < 10; i++ {
		normalCount, overCount, _ := rl.PushRequest("group1", "model1", int64(maxReq), duration, 1)

		if i < maxReq {
			if normalCount != int64(i+1) || overCount != 0 {
				t.Errorf("Request %d: expected normal=%d, over=0, got normal=%d, over=%d",
					i+1, i+1, normalCount, overCount)
			}
		} else {
			expectedOver := int64(i - maxReq)
			if normalCount != int64(maxReq+1) || overCount != expectedOver {
				t.Errorf("Request %d: expected normal=5, over=%d, got normal=%d, over=%d",
					i+1, expectedOver, normalCount, overCount)
			}
		}
	}
}

func TestEmptyQueries(t *testing.T) {
	rl := reqlimit.NewInMemoryRecord()

	totalCount, secondCount := rl.GetRequest("*", "*", 60*time.Second)
	if totalCount != 0 || secondCount != 0 {
		t.Errorf("Expected empty results, got total=%d, second=%d", totalCount, secondCount)
	}

	totalCount, secondCount = rl.GetRequest("nonexistent", "model", 60*time.Second)
	if totalCount != 0 || secondCount != 0 {
		t.Errorf("Expected empty results for nonexistent key, got total=%d, second=%d", totalCount, secondCount)
	}
}

func BenchmarkPushRequest(b *testing.B) {
	rl := reqlimit.NewInMemoryRecord()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			group := fmt.Sprintf("group%d", i%10)
			model := fmt.Sprintf("model%d", i%5)
			rl.PushRequest(group, model, 100, 60*time.Second, 1)
			i++
		}
	})
}

func BenchmarkGetRequest(b *testing.B) {
	rl := reqlimit.NewInMemoryRecord()

	for i := 0; i < 100; i++ {
		group := fmt.Sprintf("group%d", i%10)
		model := fmt.Sprintf("model%d", i%5)
		rl.PushRequest(group, model, 100, 60*time.Second, 1)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			group := fmt.Sprintf("group%d", i%10)
			model := fmt.Sprintf("model%d", i%5)
			rl.GetRequest(group, model, 60*time.Second)
			i++
		}
	})
}
