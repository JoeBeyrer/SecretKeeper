package main

import (
	"secret-keeper-app/backend/messaging"
	"testing"
)

func Test_hub_register_and_send(t *testing.T) {
	hub := messaging.NewHub()

	client := &messaging.Client{
		UserID: "user-1",
		Send:   make(chan []byte, 10),
	}

	hub.Register(client)

	msg := []byte("hello from hub")
	hub.SendToUser("user-1", msg)

	select {
	case received := <-client.Send:
		if string(received) != string(msg) {
			t.Fatalf("expected '%s', got '%s'", msg, received)
		}
		t.Log("hub correctly delivered message to registered client")
	default:
		t.Fatal("expected message in client send channel, got none")
	}
}

func Test_hub_send_to_unregistered_user(t *testing.T) {
	hub := messaging.NewHub()

	// Sending to a user who is not registered should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("hub.SendToUser panicked for unregistered user: %v", r)
		}
	}()

	hub.SendToUser("ghost-user", []byte("this should be dropped silently"))
	t.Log("send to unregistered user did not panic")
}

func Test_hub_unregister(t *testing.T) {
	hub := messaging.NewHub()

	client := &messaging.Client{
		UserID: "user-2",
		Send:   make(chan []byte, 10),
	}

	hub.Register(client)
	hub.Unregister("user-2")

	// After unregistering, messages sent to this user should be silently dropped
	hub.SendToUser("user-2", []byte("should not arrive"))

	select {
	case msg := <-client.Send:
		t.Fatalf("expected no message after unregister, but received: %s", msg)
	default:
		t.Log("no message received after unregister — correct")
	}
}

func Test_hub_register_replaces_existing_client(t *testing.T) {
	hub := messaging.NewHub()

	oldSend := make(chan []byte, 10)
	newSend := make(chan []byte, 10)

	old := &messaging.Client{UserID: "user-3", Send: oldSend}
	new := &messaging.Client{UserID: "user-3", Send: newSend}

	hub.Register(old)
	hub.Register(new) // re-registration with same userID

	hub.SendToUser("user-3", []byte("ping"))

	select {
	case msg := <-newSend:
		if string(msg) != "ping" {
			t.Fatalf("expected 'ping', got '%s'", msg)
		}
		t.Log("re-registration correctly replaced old client")
	default:
		t.Fatal("expected message on new client, got none")
	}

	// Old client should not have received anything
	select {
	case msg := <-oldSend:
		t.Fatalf("old client unexpectedly received message: %s", msg)
	default:
		t.Log("old client correctly received nothing after re-registration")
	}
}

func Test_hub_multiple_clients(t *testing.T) {
	hub := messaging.NewHub()

	users := []string{"user-a", "user-b", "user-c"}
	clients := map[string]*messaging.Client{}

	for _, uid := range users {
		c := &messaging.Client{UserID: uid, Send: make(chan []byte, 10)}
		clients[uid] = c
		hub.Register(c)
	}

	// Send to each user individually and verify only that user receives it
	for _, uid := range users {
		msg := []byte("msg-for-" + uid)
		hub.SendToUser(uid, msg)
	}

	for _, uid := range users {
		select {
		case received := <-clients[uid].Send:
			expected := "msg-for-" + uid
			if string(received) != expected {
				t.Fatalf("user %s: expected '%s', got '%s'", uid, expected, received)
			}
		default:
			t.Fatalf("user %s did not receive their message", uid)
		}
	}
	t.Log("all clients received their individual messages correctly")
}
