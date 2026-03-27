package domain

import "time"

type AnalysisResult struct {
	ProposalID   string    `json:"proposal_id"`
	AnalysisType string    `json:"analysis_type"`
	Result       string    `json:"result"`
	Provider     string    `json:"provider"`
	Score        int       `json:"score"`
	Reason       string    `json:"reason"`
	CreatedAt    time.Time `json:"created_at"`
}

func AnalyzeDocuments(proposalID string, documents []Document, now time.Time) AnalysisResult {
	result := AnalysisResult{
		ProposalID:   proposalID,
		AnalysisType: "document",
		Provider:     "mock-document-engine",
		Score:        780,
		Reason:       "documentos validados no fluxo simulado",
		CreatedAt:    now.UTC(),
	}

	if len(documents) == 0 {
		result.Result = "awaiting_additional_documents"
		result.Score = 0
		result.Reason = "nenhum documento recebido"
		return result
	}

	for _, document := range documents {
		if document.Status != StatusUploaded {
			result.Result = "awaiting_additional_documents"
			result.Score = 200
			result.Reason = "ha documentos aguardando envio completo"
			return result
		}

		switch {
		case containsKeyword(document.FileName, "ilegivel"), containsKeyword(document.FileName, "pendente"):
			result.Result = "awaiting_additional_documents"
			result.Score = 320
			result.Reason = "documento precisa ser reenviado"
			return result
		case containsKeyword(document.FileName, "manual"):
			result.Result = "manual_review"
			result.Score = 510
			result.Reason = "documento direcionado para revisao manual"
			return result
		}
	}

	result.Result = "approved"
	return result
}
