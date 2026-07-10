package events

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBroker_PublishReachesSubscriber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(ServeSSE))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	for i := 0; i < 100; i++ {
		if ActiveSessions.Load() > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.Equal(t, int64(1), ActiveSessions.Load())

	Server.Publish([]byte(`{"hello":"world"}`))

	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	require.NoError(t, err)
	assert.Equal(t, "data: {\"hello\":\"world\"}\n", line)

	resp.Body.Close()
	for i := 0; i < 100; i++ {
		if ActiveSessions.Load() == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.Equal(t, int64(0), ActiveSessions.Load())
	assert.Equal(t, int64(1), SessionsSeen.Load())
}

func TestBroker_PublishWithoutSubscribersDoesNotBlock(t *testing.T) {
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			Server.Publish([]byte("orphaned"))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked with no subscribers")
	}
}

func TestBroker_StuckSubscriberDoesNotBlockPublish(t *testing.T) {
	ch := Server.subscribe()
	defer Server.unsubscribe(ch)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			Server.Publish([]byte("flood"))
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked on a subscriber that never drains")
	}
}
