package httpapi

import (
	"strings"
	"testing"
)

func TestFallbackClassificationParsesExplicitZoneSignal(t *testing.T) {
	message := "XAUUSD BUY 3768/3763\nTP1. 3771\nTP2. 3775\nTP3. 3779\nTP4. 3783\nTP5. 3787\nTP6. 3791\nSL. 3756\nUse Money Management."
	classification := fallbackClassification(message)

	if classification.Type != "NEW_SIGNAL" || !classification.IsSignal {
		t.Fatalf("classification = %#v, want a new signal", classification)
	}
	if !parserStructuredEnough(message, classification) {
		t.Fatalf("explicit signal was not considered structured: %#v", classification)
	}
	if classification.Action == nil || *classification.Action != "BUY" {
		t.Fatalf("action = %v, want BUY", classification.Action)
	}
	if classification.Symbol == nil || *classification.Symbol != "XAUUSD" {
		t.Fatalf("symbol = %v, want XAUUSD", classification.Symbol)
	}
	if classification.EntryLow == nil || *classification.EntryLow != 3763 || classification.EntryHigh == nil || *classification.EntryHigh != 3768 {
		t.Fatalf("entry zone = [%v, %v], want [3763, 3768]", classification.EntryLow, classification.EntryHigh)
	}
	if len(classification.TP) != 6 || classification.TP[0] != 3771 || classification.TP[5] != 3791 {
		t.Fatalf("take profits = %v, want six targets from 3771 through 3791", classification.TP)
	}
	if classification.SL == nil || *classification.SL != 3756 {
		t.Fatalf("stop loss = %v, want 3756", classification.SL)
	}
}

func TestNewTypeAliasAndEnumCasingValidateAsPending(t *testing.T) {
	action, orderType, symbol := " buy ", " auto ", " xauusd "
	classification := normalizeParserClassification(signalClassification{
		IsSignal: true, Type: " new ", Action: &action, OrderType: &orderType, Symbol: &symbol,
		EntryLow: floatPtr(3763), EntryHigh: floatPtr(3768),
		TP: []float64{3771, 3775, 3779, 3783, 3787, 3791}, SL: floatPtr(3756),
		Confidence: 1, Reason: "Explicit buy signal with entry zone and targets.",
	})

	if classification.Type != "NEW_SIGNAL" || *classification.Action != "BUY" || *classification.OrderType != "AUTO" || *classification.Symbol != "XAUUSD" {
		t.Fatalf("classification was not canonicalized: %#v", classification)
	}
	classification = preferFarthestTelegramTarget(classification, "PARTIAL")
	if len(classification.TP) != 6 {
		t.Fatalf("valid partial plan lost TP targets: %v", classification.TP)
	}
	settings := parserSettings{threshold: .75, mapping: map[string]string{"XAUUSD": "XAUUSD"}}
	status, resolvedSymbol, normalized, reason := validateParserClassification(classification, settings)
	if status != "PENDING" {
		t.Fatalf("valid explicit signal status = %q, reason %q; want PENDING", status, reason)
	}
	if resolvedSymbol == nil || *resolvedSymbol != "XAUUSD" || normalized == nil || !strings.Contains(*normalized, "3763") {
		t.Fatalf("resolved signal = symbol %v, normalized %v", resolvedSymbol, normalized)
	}
}

func TestSingleEntryModeKeepsRiskValidFarthestTarget(t *testing.T) {
	action, orderType, symbol := "BUY", "AUTO", "XAUUSD"
	classification := signalClassification{
		IsSignal: true, Type: "NEW_SIGNAL", Action: &action, OrderType: &orderType, Symbol: &symbol,
		EntryLow: floatPtr(3763), EntryHigh: floatPtr(3768),
		TP: []float64{3771, 3775, 3779, 3783, 3787, 3791}, SL: floatPtr(3756),
		Confidence: 1,
	}
	classification = preferFarthestTelegramTarget(classification, "SINGLE")
	if len(classification.TP) != 1 || classification.TP[0] != 3791 {
		t.Fatalf("single-entry target = %v, want farthest risk-valid target 3791", classification.TP)
	}
}

func TestUnknownSignalTypeRemainsRejected(t *testing.T) {
	classification := signalClassification{IsSignal: true, Type: "SOMETHING_ELSE", Confidence: 1}
	status, _, _, reason := validateParserClassification(classification, parserSettings{threshold: .75})
	if status != "REJECTED" || reason != "Jenis signal tidak dikenal" {
		t.Fatalf("unknown type status/reason = %q/%q", status, reason)
	}
}
