package main

import (
	"testing"
)

func TestRunEvaluationLogic_FlagNil(t *testing.T) {
	app := &App{}
	info := &CombinedFlagInfo{Flag: nil, Rule: nil}
	result := app.runEvaluationLogic(info, "user1")
	if result != false {
		t.Error("esperado false quando flag é nil")
	}
}

func TestRunEvaluationLogic_FlagDisabled(t *testing.T) {
	app := &App{}
	info := &CombinedFlagInfo{
		Flag: &Flag{Name: "test", IsEnabled: false},
		Rule: nil,
	}
	result := app.runEvaluationLogic(info, "user1")
	if result != false {
		t.Error("esperado false quando flag está desabilitada")
	}
}

func TestRunEvaluationLogic_FlagEnabledNoRule(t *testing.T) {
	app := &App{}
	info := &CombinedFlagInfo{
		Flag: &Flag{Name: "test", IsEnabled: true},
		Rule: nil,
	}
	result := app.runEvaluationLogic(info, "user1")
	if result != true {
		t.Error("esperado true quando flag habilitada e sem regra")
	}
}

func TestRunEvaluationLogic_FlagEnabledRuleDisabled(t *testing.T) {
	app := &App{}
	info := &CombinedFlagInfo{
		Flag: &Flag{Name: "test", IsEnabled: true},
		Rule: &TargetingRule{IsEnabled: false},
	}
	result := app.runEvaluationLogic(info, "user1")
	if result != true {
		t.Error("esperado true quando flag habilitada e regra desabilitada")
	}
}

func TestRunEvaluationLogic_Percentage100(t *testing.T) {
	app := &App{}
	info := &CombinedFlagInfo{
		Flag: &Flag{Name: "test", IsEnabled: true},
		Rule: &TargetingRule{
			IsEnabled: true,
			Rules:     Rule{Type: "PERCENTAGE", Value: float64(100)},
		},
	}
	result := app.runEvaluationLogic(info, "user1")
	if result != true {
		t.Error("esperado true com porcentagem 100%")
	}
}

func TestRunEvaluationLogic_Percentage0(t *testing.T) {
	app := &App{}
	info := &CombinedFlagInfo{
		Flag: &Flag{Name: "test", IsEnabled: true},
		Rule: &TargetingRule{
			IsEnabled: true,
			Rules:     Rule{Type: "PERCENTAGE", Value: float64(0)},
		},
	}
	result := app.runEvaluationLogic(info, "user1")
	if result != false {
		t.Error("esperado false com porcentagem 0%")
	}
}

func TestRunEvaluationLogic_PercentageInvalidValue(t *testing.T) {
	app := &App{}
	info := &CombinedFlagInfo{
		Flag: &Flag{Name: "test", IsEnabled: true},
		Rule: &TargetingRule{
			IsEnabled: true,
			Rules:     Rule{Type: "PERCENTAGE", Value: "invalid"},
		},
	}
	result := app.runEvaluationLogic(info, "user1")
	if result != false {
		t.Error("esperado false com valor inválido de porcentagem")
	}
}

func TestGetDeterministicBucket(t *testing.T) {
	bucket := getDeterministicBucket("test_input")
	if bucket < 0 || bucket >= 100 {
		t.Errorf("bucket deveria estar entre 0-99, got: %d", bucket)
	}
}

func TestGetDeterministicBucket_Deterministic(t *testing.T) {
	bucket1 := getDeterministicBucket("user1flag1")
	bucket2 := getDeterministicBucket("user1flag1")
	if bucket1 != bucket2 {
		t.Error("bucket deveria ser determinístico para o mesmo input")
	}
}

func TestGetDeterministicBucket_Distribution(t *testing.T) {
	buckets := make(map[int]int)
	for i := 0; i < 1000; i++ {
		bucket := getDeterministicBucket(string(rune(i)) + "test")
		buckets[bucket]++
	}
	if len(buckets) < 20 {
		t.Errorf("distribuição muito concentrada, apenas %d buckets distintos", len(buckets))
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{FlagName: "my-flag"}
	if err.Error() != "flag ou regra 'my-flag' não encontrada" {
		t.Errorf("mensagem de erro inesperada: %s", err.Error())
	}
}
