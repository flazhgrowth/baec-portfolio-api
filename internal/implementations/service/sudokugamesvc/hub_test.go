package sudokugamesvc

import (
	"sync"
	"testing"
	"time"

	"github.com/flazhgrowth/baec-portfolio-api/internal/entity/sudokugame"
)

func update(version int) sudokugame.GameUpdate {
	return sudokugame.NewGameUpdate(sudokugame.GameResponse{Version: version})
}

func TestHub(t *testing.T) {
	t.Run("delivers to every stream of the game and to no other game", func(t *testing.T) {
		h := newHub()
		a, cancelA := h.subscribe("g1")
		b, cancelB := h.subscribe("g1")
		other, cancelOther := h.subscribe("g2")
		defer cancelA()
		defer cancelB()
		defer cancelOther()

		h.publish("g1", update(7))
		if (<-a).Game.Version != 7 || (<-b).Game.Version != 7 {
			t.Fatal("both g1 streams must receive the update")
		}
		select {
		case <-other:
			t.Fatal("g2 must not receive g1's update")
		default:
		}
	})

	t.Run("cancel closes the stream, is idempotent, and frees the game entry", func(t *testing.T) {
		h := newHub()
		ch, cancel := h.subscribe("g1")
		cancel()
		cancel()
		if _, open := <-ch; open {
			t.Fatal("channel should be closed")
		}
		if h.count("g1") != 0 || len(h.subs) != 0 {
			t.Fatal("hub should hold nothing for the game")
		}
		h.publish("g1", update(1)) // nobody listening: must not panic
	})

	t.Run("a stream that falls too far behind is cut, the others are unaffected", func(t *testing.T) {
		h := newHub()
		slow, cancelSlow := h.subscribe("g1")
		fast, cancelFast := h.subscribe("g1")
		defer cancelSlow()
		defer cancelFast()

		done := make(chan struct{})
		go func() {
			defer close(done)
			for i := 1; i <= subscriberBuffer+5; i++ {
				h.publish("g1", update(i)) // must never block on the slow stream
				<-fast                     // the fast stream keeps up
			}
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("publish blocked on a slow subscriber")
		}

		received := 0
		for range slow { // drains what was buffered, then sees the close
			received++
		}
		if received != subscriberBuffer {
			t.Fatalf("slow stream should have got its buffer's worth then been closed, got %d", received)
		}
		if h.count("g1") != 1 {
			t.Fatalf("only the fast stream should remain, have %d", h.count("g1"))
		}
	})

	t.Run("concurrent subscribe, publish and cancel are race-free", func(t *testing.T) {
		h := newHub()
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				ch, cancel := h.subscribe("g1")
				for range 3 {
					select {
					case <-ch:
					case <-time.After(time.Millisecond):
					}
				}
				cancel()
			}()
			go func(v int) {
				defer wg.Done()
				for range 50 {
					h.publish("g1", update(v))
				}
			}(i)
		}
		wg.Wait()
		if h.count("g1") != 0 {
			t.Fatal("all streams were cancelled")
		}
	})
}
