package activity

import (
	"errors"
	"testing"
)

func TestResolveInteractionChannelsFromStatuses(t *testing.T) {
	t.Run("call only", func(t *testing.T) {
		kind, call, chat, err := resolveInteractionChannels("", "TERHUBUNG", "")
		if err != nil {
			t.Fatalf("resolveInteractionChannels: %v", err)
		}
		if kind != InteractionCall {
			t.Fatalf("kind = %s, want %s", kind, InteractionCall)
		}
		if !call.Valid || call.String != "TERHUBUNG" {
			t.Fatalf("call status = %#v, want valid TERHUBUNG", call)
		}
		if chat.Valid {
			t.Fatalf("chat status = %#v, want null", chat)
		}
	})

	t.Run("chat only", func(t *testing.T) {
		kind, call, chat, err := resolveInteractionChannels("", "", "TERBALAS")
		if err != nil {
			t.Fatalf("resolveInteractionChannels: %v", err)
		}
		if kind != InteractionChat {
			t.Fatalf("kind = %s, want %s", kind, InteractionChat)
		}
		if call.Valid {
			t.Fatalf("call status = %#v, want null", call)
		}
		if !chat.Valid || chat.String != "TERBALAS" {
			t.Fatalf("chat status = %#v, want valid TERBALAS", chat)
		}
	})

	t.Run("call and chat", func(t *testing.T) {
		kind, call, chat, err := resolveInteractionChannels("", "TERHUBUNG", "TERBALAS")
		if err != nil {
			t.Fatalf("resolveInteractionChannels: %v", err)
		}
		if kind != InteractionCallChat {
			t.Fatalf("kind = %s, want %s", kind, InteractionCallChat)
		}
		if !call.Valid || !chat.Valid {
			t.Fatalf("call/chat should both be valid, got call=%#v chat=%#v", call, chat)
		}
	})
}

func TestResolveInteractionChannelsLegacyFallback(t *testing.T) {
	kind, call, chat, err := resolveInteractionChannels("CALL", "", "")
	if err != nil {
		t.Fatalf("resolveInteractionChannels legacy: %v", err)
	}
	if kind != InteractionCall {
		t.Fatalf("kind = %s, want %s", kind, InteractionCall)
	}
	if !call.Valid || call.String != "RECORDED" {
		t.Fatalf("call status = %#v, want RECORDED", call)
	}
	if chat.Valid {
		t.Fatalf("chat status = %#v, want null", chat)
	}
}

func TestResolveInteractionChannelsRejectsEmptyInput(t *testing.T) {
	_, _, _, err := resolveInteractionChannels("", "", "")
	if !errors.Is(err, ErrInteractionEmpty) {
		t.Fatalf("err = %v, want ErrInteractionEmpty", err)
	}
}
