package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

func TestAdmissionHandler_ServeHTTP(t *testing.T) {
	t.Run("successful admission", func(t *testing.T) {
		handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
			return &admissionv1.AdmissionResponse{
				Allowed: true,
			}
		})

		review := createAdmissionReview("test-uid", nil)
		body, _ := json.Marshal(review)

		req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp admissionv1.AdmissionReview
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !resp.Response.Allowed {
			t.Error("Expected Allowed=true")
		}
		if resp.Response.UID != "test-uid" {
			t.Errorf("Expected UID %q, got %q", "test-uid", resp.Response.UID)
		}
	})

	t.Run("denied admission", func(t *testing.T) {
		handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
			return &admissionv1.AdmissionResponse{
				Allowed: false,
				Result: &metav1.Status{
					Message: "denied",
				},
			}
		})

		review := createAdmissionReview("test-uid", nil)
		body, _ := json.Marshal(review)

		req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp admissionv1.AdmissionReview
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Response.Allowed {
			t.Error("Expected Allowed=false")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
			return &admissionv1.AdmissionResponse{Allowed: true}
		})

		req := httptest.NewRequest(http.MethodPost, "/mutate", nil)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("wrong content type", func(t *testing.T) {
		handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
			return &admissionv1.AdmissionResponse{Allowed: true}
		})

		req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader([]byte("test")))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Expected status %d, got %d", http.StatusUnsupportedMediaType, rec.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
			return &admissionv1.AdmissionResponse{Allowed: true}
		})

		req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader([]byte("{invalid json}")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("nil request in review", func(t *testing.T) {
		handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
			return &admissionv1.AdmissionResponse{Allowed: true}
		})

		review := admissionv1.AdmissionReview{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "admission.k8s.io/v1",
				Kind:       "AdmissionReview",
			},
			Request: nil,
		}
		body, _ := json.Marshal(review)

		req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rec.Code)
		}

		var resp admissionv1.AdmissionReview
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.Response.Result == nil || resp.Response.Result.Code != http.StatusBadRequest {
			t.Error("Expected bad request response for nil Request")
		}
	})

	t.Run("preserves API version", func(t *testing.T) {
		handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
			return &admissionv1.AdmissionResponse{Allowed: true}
		})

		review := createAdmissionReview("test-uid", nil)
		review.APIVersion = "admission.k8s.io/v1"
		body, _ := json.Marshal(review)

		req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		var resp admissionv1.AdmissionReview
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if resp.APIVersion != "admission.k8s.io/v1" {
			t.Errorf("Expected APIVersion %q, got %q", "admission.k8s.io/v1", resp.APIVersion)
		}
	})
}

func TestAdmissionHandler_WithPatch(t *testing.T) {
	patchType := admissionv1.PatchTypeJSONPatch
	handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
		return &admissionv1.AdmissionResponse{
			Allowed:   true,
			Patch:     []byte(`[{"op":"add","path":"/metadata/labels/test","value":"true"}]`),
			PatchType: &patchType,
		}
	})

	review := createAdmissionReview("test-uid", []byte(`{"metadata":{"name":"test"}}`))
	body, _ := json.Marshal(review)

	req := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !resp.Response.Allowed {
		t.Error("Expected Allowed=true")
	}
	if resp.Response.Patch == nil {
		t.Error("Expected non-nil Patch")
	}
	if resp.Response.PatchType == nil || *resp.Response.PatchType != admissionv1.PatchTypeJSONPatch {
		t.Error("Expected PatchType=JSONPatch")
	}
}

// Helper to create an AdmissionReview
func createAdmissionReview(uid string, object []byte) admissionv1.AdmissionReview {
	review := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: types.UID(uid),
			Kind: metav1.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			Resource: metav1.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			Name:      "test-pod",
			Namespace: "default",
			Operation: admissionv1.Create,
		},
	}

	if object != nil {
		review.Request.Object = runtime.RawExtension{Raw: object}
	}

	return review
}

// Test error reading body
func TestAdmissionHandler_ReadBodyError(t *testing.T) {
	handler := newAdmissionHandler(func(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
		return &admissionv1.AdmissionResponse{Allowed: true}
	})

	// Create a reader that returns an error
	errReader := &errorReader{err: io.ErrUnexpectedEOF}
	req := httptest.NewRequest(http.MethodPost, "/mutate", errReader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
