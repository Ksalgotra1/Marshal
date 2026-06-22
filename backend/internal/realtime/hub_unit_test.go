package realtime

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type HubUnitSuite struct {
	suite.Suite
	hub *Hub
}

func TestHubUnitSuite(t *testing.T) {
	suite.Run(t, new(HubUnitSuite))
}

func (s *HubUnitSuite) SetupTest() {
	s.hub = NewHub()
	go s.hub.Run()
}

func (s *HubUnitSuite) TestStreamClientReceivesRoomBroadcastAndConnectionCount() {
	client := newFakeStream([]string{"global", "group-1"}, 4)
	s.hub.RegisterStreamClient(client)

	s.Eventually(func() bool { return s.hub.ConnectionCount() == 1 }, time.Second, 10*time.Millisecond)
	s.hub.Broadcast("group-1", GroupFormed("group-1", 3, 42))

	event := client.readEvent(s.T())
	if event.Type == "system:connections" {
		event = client.readEvent(s.T())
	}
	s.Equal("group:formed", event.Type)
	s.Equal("group-1", event.GroupID)
}

func (s *HubUnitSuite) TestBroadcastMultiTargetsEachRoom() {
	global := newFakeStream([]string{"global"}, 4)
	group := newFakeStream([]string{"group-2"}, 4)
	s.hub.RegisterStreamClient(global)
	s.hub.RegisterStreamClient(group)
	s.Eventually(func() bool { return s.hub.ConnectionCount() == 2 }, time.Second, 10*time.Millisecond)

	s.hub.BroadcastMulti([]string{"global", "group-2"}, GroupDispatching("group-2", 1, 77))

	s.Equal("group:dispatching", drainUntilType(s.T(), global, "group:dispatching").Type)
	s.Equal("group:dispatching", drainUntilType(s.T(), group, "group:dispatching").Type)
}

func (s *HubUnitSuite) TestSlowStreamClientIsEvicted() {
	client := newFakeStream([]string{"slow-room"}, 0)
	s.hub.RegisterStreamClient(client)
	s.Eventually(func() bool { return s.hub.ConnectionCount() == 1 }, time.Second, 10*time.Millisecond)

	s.hub.Broadcast("slow-room", GroupCancelled("slow-room", "timeout"))

	s.Eventually(client.isClosed, time.Second, 10*time.Millisecond)
	s.Eventually(func() bool { return s.hub.ConnectionCount() == 0 }, time.Second, 10*time.Millisecond)
}

func (s *HubUnitSuite) TestSplitRoomsDefaultsAndDropsEmptySegments() {
	s.Equal([]string{"global"}, splitRooms(""))
	s.Equal([]string{"global", "group-1"}, splitRooms("global,,group-1,"))
}

type fakeStream struct {
	rooms  []string
	send   chan []byte
	mu     sync.Mutex
	closed bool
}

func newFakeStream(rooms []string, buffer int) *fakeStream {
	return &fakeStream{rooms: rooms, send: make(chan []byte, buffer)}
}

func (f *fakeStream) Rooms() []string {
	return f.rooms
}

func (f *fakeStream) Send() chan []byte {
	return f.send
}

func (f *fakeStream) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
}

func (f *fakeStream) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func (f *fakeStream) readEvent(t *testing.T) Event {
	t.Helper()
	select {
	case raw := <-f.send:
		var event Event
		if err := json.Unmarshal(raw, &event); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}

func drainUntilType(t *testing.T, client *fakeStream, eventType string) Event {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case raw := <-client.send:
			var event Event
			if err := json.Unmarshal(raw, &event); err != nil {
				t.Fatalf("unmarshal event: %v", err)
			}
			if event.Type == eventType {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event type %s", eventType)
			return Event{}
		}
	}
}

func (s *HubUnitSuite) TestConcurrentConnectsAndDisconnects() {
	var wg sync.WaitGroup
	numClients := 50

	// Concurrent registers
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := newFakeStream([]string{"global", "concurrent-room"}, 10)
			s.hub.RegisterStreamClient(client)
			
			// Simulate a brief connection
			time.Sleep(5 * time.Millisecond)
			
			// Concurrent unregisters
			s.hub.UnregisterStreamClient(client)
		}(i)
	}

	wg.Wait()
	
	// Ensure connection count goes back to 0 cleanly without panics or races
	s.Eventually(func() bool { return s.hub.ConnectionCount() == 0 }, time.Second, 10*time.Millisecond)
}

func (s *HubUnitSuite) TestGhostRoomBroadcast() {
	// Broadcast to a room that has never been created or joined
	s.NotPanics(func() {
		s.hub.Broadcast("ghost-room", GroupFormed("ghost-room", 4, 99.9))
	})
	
	// Broadcast multiple to ghost rooms
	s.NotPanics(func() {
		s.hub.BroadcastMulti([]string{"ghost-1", "ghost-2"}, GroupCancelled("ghost-1", "test"))
	})
}
