package ai

import (
	"strings"
	"testing"
)

func TestValidateResponse(t *testing.T) {
	req := []BlockRequest{
		{ID: "b1", Text: "Hello world."},
		{ID: "b2", Text: "Second paragraph."},
	}

	// Valid response
	validResp := []BlockResponse{
		{ID: "b1", Text: "Hello world, edited."},
		{ID: "b2", Text: "Second paragraph, edited."},
	}
	if err := ValidateResponse(req, validResp); err != nil {
		t.Fatalf("expected valid response to pass, got: %v", err)
	}

	// Missing ID
	missingResp := []BlockResponse{
		{ID: "b1", Text: "Hello world, edited."},
	}
	if err := ValidateResponse(req, missingResp); err == nil {
		t.Errorf("expected error for missing ID, got nil")
	}

	// Extra/mismatched ID
	mismatchResp := []BlockResponse{
		{ID: "b1", Text: "Hello world, edited."},
		{ID: "b3", Text: "Different block."},
	}
	if err := ValidateResponse(req, mismatchResp); err == nil {
		t.Errorf("expected error for mismatched ID, got nil")
	}

	// Empty text
	emptyResp := []BlockResponse{
		{ID: "b1", Text: ""},
		{ID: "b2", Text: "Non-empty."},
	}
	if err := ValidateResponse(req, emptyResp); err == nil {
		t.Errorf("expected error for empty text, got nil")
	}
}

func TestHighlightDensityAndSanitizer(t *testing.T) {
	normalText := "This is a regular paragraph with one ==important== keyword and no excess markup."
	exceeds, density := CheckHighlightDensity(normalText)
	if exceeds {
		t.Errorf("normalText should not exceed density limit (density=%f)", density)
	}

	overHighlighted := "==This== ==is== ==a== ==very== ==heavily== ==highlighted== paragraph with lots of highlights."
	exceedsOver, densityOver := CheckHighlightDensity(overHighlighted)
	if !exceedsOver {
		t.Errorf("overHighlighted should exceed density limit (density=%f)", densityOver)
	}

	sanitized := SanitizeHighlights(overHighlighted, 0.15)
	exceedsAfter, _ := CheckHighlightDensity(sanitized)
	if exceedsAfter {
		t.Errorf("sanitized text should not exceed density threshold")
	}
}

func TestComputeEditDistance(t *testing.T) {
	orig := "The quick brown fox jumps over the lazy dog"
	same := "The quick brown fox jumps over the lazy dog"
	distZero := ComputeEditDistance(orig, same)
	if distZero != 0.0 {
		t.Errorf("expected distance 0.0 for identical strings, got %f", distZero)
	}

	minor := "The fast brown fox jumps over the lazy dog"
	distMinor := ComputeEditDistance(orig, minor)
	if distMinor <= 0.0 || distMinor > 0.35 {
		t.Errorf("expected minor distance between 0 and 0.35, got %f", distMinor)
	}

	totallyDiff := "Something completely unrelated and different in every single word spoken"
	distHeavy := ComputeEditDistance(orig, totallyDiff)
	if distHeavy <= 0.35 {
		t.Errorf("expected heavy distance > 0.35, got %f", distHeavy)
	}
}

func TestPrepareDiff(t *testing.T) {
	req := []BlockRequest{
		{ID: "b1", Text: "First original line."},
		{ID: "b2", Text: "Completely different text here."},
	}
	resp := []BlockResponse{
		{ID: "b1", Text: "First original line."},
		{ID: "b2", Text: "Totally replaced words all over the place."},
	}

	diffs := PrepareDiff(req, resp)
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diff blocks, got %d", len(diffs))
	}

	if diffs[0].HeavilyChanged {
		t.Errorf("block 1 should not be heavily changed")
	}
	if !strings.Contains(diffs[1].Refined, "Totally replaced") {
		t.Errorf("refined text mismatch in block 2")
	}
}
