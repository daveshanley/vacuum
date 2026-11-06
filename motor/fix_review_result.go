package motor

// TakeResult transfers a verified preview's lint resources to the caller.
// It grants no write authority; callers must first obtain acceptance and check
// the original source at the write boundary. The preview can no longer be used.
func (p *FixPreview) TakeResult() *RuleSetExecutionResult {
	if p == nil || p.final == nil {
		return nil
	}
	result := p.final
	p.final = nil
	result.ModifiedSpec = append([]byte(nil), p.Candidate...)
	result.FixedResults = p.fixed
	result.RuleSetExecution.Spec = append([]byte(nil), p.original...)
	p.Candidate = nil
	return result
}

// TakeBaselineResult transfers the original lint result when no fixes are used.
// The review is closed after transfer; the caller must release the result.
func (r *FixReview) TakeBaselineResult() *RuleSetExecutionResult {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := r.baseline
	r.baseline = nil
	return result
}
