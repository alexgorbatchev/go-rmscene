package rmscene_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alexgorbatchev/go-rmscene"
)

func TestValidateHeader_Valid(t *testing.T) {
	data := []byte(rmscene.HeaderV6)
	if err := rmscene.ValidateHeader(bytes.NewReader(data)); err != nil {
		t.Fatalf("expected valid header, got error: %v", err)
	}
	if err := rmscene.ValidateHeaderBytes(data); err != nil {
		t.Fatalf("expected valid header bytes, got error: %v", err)
	}
}

func TestValidateHeader_TooShort(t *testing.T) {
	data := []byte("reMarkable .lines file, version=6")
	if err := rmscene.ValidateHeader(bytes.NewReader(data)); err == nil {
		t.Fatal("expected error for too short header, got nil")
	}
	if err := rmscene.ValidateHeaderBytes(data); err == nil {
		t.Fatal("expected error for too short header bytes, got nil")
	}
}

func TestValidateHeader_WrongMagic(t *testing.T) {
	data := []byte("reMarkable .lines file, version=5          ")
	if err := rmscene.ValidateHeader(bytes.NewReader(data)); err == nil {
		t.Fatal("expected error for v5 header, got nil")
	}
	if err := rmscene.ValidateHeaderBytes(data); err == nil {
		t.Fatal("expected error for v5 header bytes, got nil")
	}
}

func TestValidateHeader_ExtraDataValid(t *testing.T) {
	data := []byte(rmscene.HeaderV6 + "extra binary payload")
	if err := rmscene.ValidateHeader(bytes.NewReader(data)); err != nil {
		t.Fatalf("expected valid header with payload, got: %v", err)
	}
	if err := rmscene.ValidateHeaderBytes(data); err != nil {
		t.Fatalf("expected valid header bytes with payload, got: %v", err)
	}
}

func TestValidateHeader_ExactLength(t *testing.T) {
	if len(rmscene.HeaderV6) != 43 {
		t.Fatalf("expected header length 43, got %d", len(rmscene.HeaderV6))
	}
	if !strings.HasSuffix(rmscene.HeaderV6, "          ") {
		t.Fatalf("expected 10 trailing spaces in header")
	}
}
