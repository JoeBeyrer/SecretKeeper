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
	hub.Unregister("user-2", client)

	// After unregistering, messages sent to this user should be silently dropped
	hub.SendToUser("user-2", []byte("should not arrive"))

	select {
	case msg := <-client.Send:
		t.Fatalf("expected no message after unregister, but received: %s", msg)
	default:
		t.Log("no message received after unregister — correct")
	}
}

func Test_hub_register_multiple_tabs_same_user(t *testing.T) {
	hub := messaging.NewHub()

	tab1Send := make(chan []byte, 10)
	tab2Send := make(chan []byte, 10)

	tab1 := &messaging.Client{UserID: "user-3", Send: tab1Send}
	tab2 := &messaging.Client{UserID: "user-3", Send: tab2Send}

	hub.Register(tab1)
	hub.Register(tab2) // second tab for same user

	hub.SendToUser("user-3", []byte("ping"))

	// Both tabs should receive the message
	select {
	case msg := <-tab1Send:
		if string(msg) != "ping" {
			t.Fatalf("tab1: expected 'ping', got '%s'", msg)
		}
		t.Log("tab1 correctly received message")
	default:
		t.Fatal("tab1 expected message but got none")
	}

	select {
	case msg := <-tab2Send:
		if string(msg) != "ping" {
			t.Fatalf("tab2: expected 'ping', got '%s'", msg)
		}
		t.Log("tab2 correctly received message")
	default:
		t.Fatal("tab2 expected message but got none")
	}

	// Unregister tab1 only - tab2 should still receive messages
	hub.Unregister("user-3", tab1)
	hub.SendToUser("user-3", []byte("ping2"))

	select {
	case msg := <-tab2Send:
		if string(msg) != "ping2" {
			t.Fatalf("tab2: expected 'ping2', got '%s'", msg)
		}
		t.Log("tab2 still receives after tab1 unregistered")
	default:
		t.Fatal("tab2 expected message after tab1 unregistered but got none")
	}

	select {
	case msg := <-tab1Send:
		t.Fatalf("tab1 unexpectedly received message after unregister: %s", msg)
	default:
		t.Log("tab1 correctly received nothing after unregister")
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
