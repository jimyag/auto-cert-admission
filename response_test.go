package autocertwebhook

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/appscode/jsonpatch"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAllowed(t *testing.T) {
	resp := Allowed()

	if !resp.Allowed {
		t.Error("Allowed() should return Allowed=true")
	}
	if resp.Result != nil {
		t.Error("Allowed() should have nil Result")
	}
}

func TestAllowedWithMessage(t *testing.T) {
	msg := "request allowed"
	resp := AllowedWithMessage(msg)

	if !resp.Allowed {
		t.Error("AllowedWithMessage() should return Allowed=true")
	}
	if resp.Result == nil {
		t.Fatal("AllowedWithMessage() should have non-nil Result")
	}
	if resp.Result.Message != msg {
		t.Errorf("Message: got %q, want %q", resp.Result.Message, msg)
	}
}

func TestDenied(t *testing.T) {
	msg := "request denied"
	resp := Denied(msg)

	if resp.Allowed {
		t.Error("Denied() should return Allowed=false")
	}
	if resp.Result == nil {
		t.Fatal("Denied() should have non-nil Result")
	}
	if resp.Result.Status != metav1.StatusFailure {
		t.Errorf("Status: got %q, want %q", resp.Result.Status, metav1.StatusFailure)
	}
	if resp.Result.Message != msg {
		t.Errorf("Message: got %q, want %q", resp.Result.Message, msg)
	}
	if resp.Result.Reason != metav1.StatusReasonForbidden {
		t.Errorf("Reason: got %q, want %q", resp.Result.Reason, metav1.StatusReasonForbidden)
	}
	if resp.Result.Code != http.StatusForbidden {
		t.Errorf("Code: got %d, want %d", resp.Result.Code, http.StatusForbidden)
	}
}

func TestDeniedWithReason(t *testing.T) {
	msg := "invalid request"
	reason := metav1.StatusReasonBadRequest
	code := int32(http.StatusBadRequest)

	resp := DeniedWithReason(msg, reason, code)

	if resp.Allowed {
		t.Error("DeniedWithReason() should return Allowed=false")
	}
	if resp.Result == nil {
		t.Fatal("DeniedWithReason() should have non-nil Result")
	}
	if resp.Result.Message != msg {
		t.Errorf("Message: got %q, want %q", resp.Result.Message, msg)
	}
	if resp.Result.Reason != reason {
		t.Errorf("Reason: got %q, want %q", resp.Result.Reason, reason)
	}
	if resp.Result.Code != code {
		t.Errorf("Code: got %d, want %d", resp.Result.Code, code)
	}
}

func TestErrored(t *testing.T) {
	err := errors.New("something went wrong")
	resp := Errored(err)

	if resp.Allowed {
		t.Error("Errored() should return Allowed=false")
	}
	if resp.Result == nil {
		t.Fatal("Errored() should have non-nil Result")
	}
	if resp.Result.Status != metav1.StatusFailure {
		t.Errorf("Status: got %q, want %q", resp.Result.Status, metav1.StatusFailure)
	}
	if resp.Result.Message != err.Error() {
		t.Errorf("Message: got %q, want %q", resp.Result.Message, err.Error())
	}
	if resp.Result.Reason != metav1.StatusReasonInternalError {
		t.Errorf("Reason: got %q, want %q", resp.Result.Reason, metav1.StatusReasonInternalError)
	}
	if resp.Result.Code != http.StatusInternalServerError {
		t.Errorf("Code: got %d, want %d", resp.Result.Code, http.StatusInternalServerError)
	}
}

func TestErroredWithCode(t *testing.T) {
	err := errors.New("bad gateway")
	code := int32(http.StatusBadGateway)

	resp := ErroredWithCode(err, code)

	if resp.Allowed {
		t.Error("ErroredWithCode() should return Allowed=false")
	}
	if resp.Result == nil {
		t.Fatal("ErroredWithCode() should have non-nil Result")
	}
	if resp.Result.Message != err.Error() {
		t.Errorf("Message: got %q, want %q", resp.Result.Message, err.Error())
	}
	if resp.Result.Code != code {
		t.Errorf("Code: got %d, want %d", resp.Result.Code, code)
	}
}

func TestPatchResponse(t *testing.T) {
	type testObj struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels,omitempty"`
	}

	t.Run("with changes", func(t *testing.T) {
		original := testObj{Name: "test"}
		modified := testObj{Name: "test", Labels: map[string]string{"key": "value"}}

		resp := PatchResponse(original, modified)

		if !resp.Allowed {
			t.Error("PatchResponse() should return Allowed=true")
		}
		if resp.Patch == nil {
			t.Fatal("PatchResponse() should have non-nil Patch")
		}
		if resp.PatchType == nil || *resp.PatchType != admissionv1.PatchTypeJSONPatch {
			t.Error("PatchResponse() should have PatchType=JSONPatch")
		}

		// Verify patch content
		var patches []jsonpatch.JsonPatchOperation
		if err := json.Unmarshal(resp.Patch, &patches); err != nil {
			t.Fatalf("Failed to unmarshal patch: %v", err)
		}
		if len(patches) == 0 {
			t.Error("Expected non-empty patches")
		}
	})

	t.Run("no changes", func(t *testing.T) {
		original := testObj{Name: "test"}
		modified := testObj{Name: "test"}

		resp := PatchResponse(original, modified)

		if !resp.Allowed {
			t.Error("PatchResponse() should return Allowed=true")
		}
		if resp.Patch != nil {
			t.Error("PatchResponse() with no changes should have nil Patch")
		}
	})

	t.Run("unmarshalable original", func(t *testing.T) {
		// Channel cannot be marshaled to JSON
		original := make(chan int)
		modified := testObj{Name: "test"}

		resp := PatchResponse(original, modified)

		if resp.Allowed {
			t.Error("PatchResponse() with unmarshalable original should return Allowed=false")
		}
	})
}

func TestPatchResponseFromRaw(t *testing.T) {
	t.Run("with changes", func(t *testing.T) {
		original := []byte(`{"name":"test"}`)
		modified := []byte(`{"name":"test","labels":{"key":"value"}}`)

		resp := PatchResponseFromRaw(original, modified)

		if !resp.Allowed {
			t.Error("PatchResponseFromRaw() should return Allowed=true")
		}
		if resp.Patch == nil {
			t.Fatal("PatchResponseFromRaw() should have non-nil Patch")
		}
	})

	t.Run("no changes", func(t *testing.T) {
		original := []byte(`{"name":"test"}`)
		modified := []byte(`{"name":"test"}`)

		resp := PatchResponseFromRaw(original, modified)

		if !resp.Allowed {
			t.Error("PatchResponseFromRaw() should return Allowed=true")
		}
		if resp.Patch != nil {
			t.Error("PatchResponseFromRaw() with no changes should have nil Patch")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		original := []byte(`{invalid}`)
		modified := []byte(`{"name":"test"}`)

		resp := PatchResponseFromRaw(original, modified)

		if resp.Allowed {
			t.Error("PatchResponseFromRaw() with invalid JSON should return Allowed=false")
		}
	})
}

func TestPatchResponseFromPatches(t *testing.T) {
	t.Run("with patches", func(t *testing.T) {
		patches := []jsonpatch.JsonPatchOperation{
			{Operation: "add", Path: "/labels", Value: map[string]string{"key": "value"}},
		}

		resp := PatchResponseFromPatches(patches)

		if !resp.Allowed {
			t.Error("PatchResponseFromPatches() should return Allowed=true")
		}
		if resp.Patch == nil {
			t.Fatal("PatchResponseFromPatches() should have non-nil Patch")
		}
		if resp.PatchType == nil || *resp.PatchType != admissionv1.PatchTypeJSONPatch {
			t.Error("PatchResponseFromPatches() should have PatchType=JSONPatch")
		}
	})

	t.Run("empty patches", func(t *testing.T) {
		var patches []jsonpatch.JsonPatchOperation

		resp := PatchResponseFromPatches(patches)

		if !resp.Allowed {
			t.Error("PatchResponseFromPatches() should return Allowed=true")
		}
		if resp.Patch != nil {
			t.Error("PatchResponseFromPatches() with empty patches should have nil Patch")
		}
	})

	t.Run("nil patches", func(t *testing.T) {
		resp := PatchResponseFromPatches(nil)

		if !resp.Allowed {
			t.Error("PatchResponseFromPatches() should return Allowed=true")
		}
		if resp.Patch != nil {
			t.Error("PatchResponseFromPatches() with nil patches should have nil Patch")
		}
	})
}

func TestPatchResponse_UnmarshalableModified(t *testing.T) {
	type testObj struct {
		Name string `json:"name"`
	}

	// Channel cannot be marshaled to JSON
	original := testObj{Name: "test"}
	modified := make(chan int)

	resp := PatchResponse(original, modified)

	if resp.Allowed {
		t.Error("PatchResponse() with unmarshalable modified should return Allowed=false")
	}
	if resp.Result == nil {
		t.Fatal("Expected non-nil Result")
	}
	if resp.Result.Code != http.StatusInternalServerError {
		t.Errorf("Expected code %d, got %d", http.StatusInternalServerError, resp.Result.Code)
	}
}
