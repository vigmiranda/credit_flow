"use client";

import { useState } from "react";

import {
  confirmDocumentReceived,
  createDocumentUploadUrl,
  createProposal,
  getProposal,
  upsertCustomer,
  type Document,
  type Proposal,
} from "@/lib/api";

const initialCustomerForm = {
  full_name: "",
  cpf: "",
  birth_date: "",
  email: "",
  phone: "",
  monthly_income: "4500",
  address: "",
};

const initialDocumentForm = {
  document_type: "id_front",
  file_name: "documento-frente.jpg",
  content_type: "image/jpeg",
};

const statusLabels: Record<string, string> = {
  created: "Criada",
  customer_data_pending: "Cadastro pendente",
  documents_pending: "Documentos pendentes",
  documents_received: "Documentos recebidos",
  document_analysis_in_progress: "Analise documental",
  credit_analysis_in_progress: "Analise de credito",
  fraud_analysis_in_progress: "Analise antifraude",
  manual_review: "Revisao manual",
  approved: "Aprovada",
  rejected: "Reprovada",
  awaiting_additional_documents: "Complementacao de documentos",
};

const analysisLabels: Record<string, string> = {
  document: "Documentos",
  credit: "Credito",
  fraud: "Fraude",
};

const timelineLabels: Record<string, string> = {
  created: "Proposta criada",
  customer_data_pending: "Aguardando dados do cliente",
  documents_pending: "Aguardando documentos",
  documents_received: "Documentos recebidos",
  document_analysis_in_progress: "Analise documental iniciada",
  credit_analysis_in_progress: "Analise de credito iniciada",
  fraud_analysis_in_progress: "Analise antifraude iniciada",
  manual_review: "Proposta em revisao manual",
  approved: "Proposta aprovada",
  rejected: "Proposta reprovada",
  awaiting_additional_documents: "Complementacao de documentos",
};

function formatStatus(status?: string) {
  if (!status) {
    return "Nao iniciado";
  }
  return statusLabels[status] ?? status;
}

export default function HomePage() {
  const [proposal, setProposal] = useState<Proposal | null>(null);
  const [latestDocument, setLatestDocument] = useState<Document | null>(null);
  const [customerForm, setCustomerForm] = useState(initialCustomerForm);
  const [documentForm, setDocumentForm] = useState(initialDocumentForm);
  const [message, setMessage] = useState("Pronto para iniciar a jornada do MVP.");
  const [pendingAction, setPendingAction] = useState<string | null>(null);

  async function syncProposal(proposalId: string) {
    const data = await getProposal(proposalId);
    setProposal(data);
  }

  async function handleCreateProposal() {
    setPendingAction("create-proposal");
    setMessage("Criando proposta...");

    try {
      const created = await createProposal();
      await syncProposal(created.proposal_id);
      setMessage(`Proposta ${created.protocol} criada com sucesso.`);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Falha ao criar proposta.");
    } finally {
      setPendingAction(null);
    }
  }

  async function handleSaveCustomer() {
    if (!proposal) {
      return;
    }

    setPendingAction("save-customer");
    setMessage("Salvando dados do cliente...");

    try {
      await upsertCustomer(proposal.proposal_id, {
        ...customerForm,
        monthly_income: Number(customerForm.monthly_income),
      });
      await syncProposal(proposal.proposal_id);
      setMessage("Cadastro do cliente salvo.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Falha ao salvar cliente.");
    } finally {
      setPendingAction(null);
    }
  }

  async function handleCreateUploadUrl() {
    if (!proposal) {
      return;
    }

    setPendingAction("create-upload");
    setMessage("Gerando URL de upload...");

    try {
      const document = await createDocumentUploadUrl(proposal.proposal_id, documentForm);
      setLatestDocument(document);
      await syncProposal(proposal.proposal_id);
      setMessage("Documento registrado. Falta confirmar o recebimento.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Falha ao gerar upload.");
    } finally {
      setPendingAction(null);
    }
  }

  async function handleConfirmDocument(documentId: string) {
    if (!proposal) {
      return;
    }

    setPendingAction(`confirm-${documentId}`);
    setMessage("Confirmando recebimento do documento...");

    try {
      const document = await confirmDocumentReceived(proposal.proposal_id, documentId);
      setLatestDocument(document);
      await syncProposal(proposal.proposal_id);
      setMessage("Documento confirmado e proposta atualizada.");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Falha ao confirmar documento.");
    } finally {
      setPendingAction(null);
    }
  }

  const documents = proposal?.documents ?? [];
  const analysisResults = proposal?.analysis_results ?? [];
  const timelineEvents = [
    ...(proposal?.status_history ?? []).map((entry) => ({
      id: entry.status_history_id,
      timestamp: entry.created_at,
      kind: "status" as const,
      title: timelineLabels[entry.status] ?? entry.status,
      detail: `Origem: ${entry.source}`,
    })),
    ...(proposal?.notifications ?? []).map((entry) => ({
      id: entry.notification_id,
      timestamp: entry.created_at,
      kind: "notification" as const,
      title: `Notificacao ${entry.channel}`,
      detail: entry.message,
    })),
  ].sort((left, right) => new Date(right.timestamp).getTime() - new Date(left.timestamp).getTime());

  return (
    <main className="shell">
      <section className="hero">
        <div className="hero-copy">
          <p className="eyebrow">Credit Flow MVP</p>
          <h1>Jornada inicial para propostas de cartao, do protocolo ao documento.</h1>
          <p className="lead">
            Este corte conecta o front ao BFF e aos servicos de proposta, cliente e documento
            ja implementados.
          </p>
        </div>
        <div className="hero-panel">
          <span className="hero-panel-label">Status atual</span>
          <strong>{formatStatus(proposal?.status)}</strong>
          <span>{proposal?.protocol ?? "Nenhuma proposta criada ainda"}</span>
          <button
            className="primary-button"
            onClick={handleCreateProposal}
            disabled={pendingAction !== null}
          >
            {pendingAction === "create-proposal" ? "Criando..." : "Criar proposta"}
          </button>
        </div>
      </section>

      <section className="grid">
        <article className="card">
          <div className="card-header">
            <span className="step-index">01</span>
            <div>
              <h2>Proposta</h2>
              <p>Abertura e consulta consolidada da proposta.</p>
            </div>
          </div>
          <div className="summary-list">
            <div>
              <span>Proposal ID</span>
              <strong>{proposal?.proposal_id ?? "Aguardando criacao"}</strong>
            </div>
            <div>
              <span>Protocolo</span>
              <strong>{proposal?.protocol ?? "-"}</strong>
            </div>
            <div>
              <span>Status</span>
              <strong>{formatStatus(proposal?.status)}</strong>
            </div>
          </div>
          <button
            className="ghost-button"
            onClick={() => proposal && syncProposal(proposal.proposal_id)}
            disabled={!proposal || pendingAction !== null}
          >
            Atualizar proposta
          </button>
        </article>

        <article className="card">
          <div className="card-header">
            <span className="step-index">02</span>
            <div>
              <h2>Cliente</h2>
              <p>Cadastro basico com validacoes essenciais.</p>
            </div>
          </div>
          <div className="form-grid">
            <label>
              Nome completo
              <input
                value={customerForm.full_name}
                onChange={(event) =>
                  setCustomerForm((current) => ({ ...current, full_name: event.target.value }))
                }
                placeholder="Maria Silva"
              />
            </label>
            <label>
              CPF
              <input
                value={customerForm.cpf}
                onChange={(event) =>
                  setCustomerForm((current) => ({ ...current, cpf: event.target.value }))
                }
                placeholder="12345678901"
              />
            </label>
            <label>
              Nascimento
              <input
                type="date"
                value={customerForm.birth_date}
                onChange={(event) =>
                  setCustomerForm((current) => ({ ...current, birth_date: event.target.value }))
                }
              />
            </label>
            <label>
              E-mail
              <input
                type="email"
                value={customerForm.email}
                onChange={(event) =>
                  setCustomerForm((current) => ({ ...current, email: event.target.value }))
                }
                placeholder="maria@example.com"
              />
            </label>
            <label>
              Telefone
              <input
                value={customerForm.phone}
                onChange={(event) =>
                  setCustomerForm((current) => ({ ...current, phone: event.target.value }))
                }
                placeholder="11999999999"
              />
            </label>
            <label>
              Renda mensal
              <input
                type="number"
                min="0"
                value={customerForm.monthly_income}
                onChange={(event) =>
                  setCustomerForm((current) => ({
                    ...current,
                    monthly_income: event.target.value,
                  }))
                }
              />
            </label>
            <label className="full-width">
              Endereco
              <input
                value={customerForm.address}
                onChange={(event) =>
                  setCustomerForm((current) => ({ ...current, address: event.target.value }))
                }
                placeholder="Rua Exemplo, 123"
              />
            </label>
          </div>
          <button
            className="primary-button"
            onClick={handleSaveCustomer}
            disabled={!proposal || pendingAction !== null}
          >
            {pendingAction === "save-customer" ? "Salvando..." : "Salvar cliente"}
          </button>
        </article>

        <article className="card">
          <div className="card-header">
            <span className="step-index">03</span>
            <div>
              <h2>Documento</h2>
              <p>Geracao da URL e confirmacao manual do recebimento no MVP.</p>
            </div>
          </div>
          <div className="form-grid">
            <label>
              Tipo
              <select
                value={documentForm.document_type}
                onChange={(event) =>
                  setDocumentForm((current) => ({
                    ...current,
                    document_type: event.target.value,
                  }))
                }
              >
                <option value="id_front">Documento frente</option>
                <option value="id_back">Documento verso</option>
                <option value="proof_of_income">Comprovante de renda</option>
              </select>
            </label>
            <label>
              Nome do arquivo
              <input
                value={documentForm.file_name}
                onChange={(event) =>
                  setDocumentForm((current) => ({ ...current, file_name: event.target.value }))
                }
              />
            </label>
            <label>
              Content type
              <input
                value={documentForm.content_type}
                onChange={(event) =>
                  setDocumentForm((current) => ({
                    ...current,
                    content_type: event.target.value,
                  }))
                }
              />
            </label>
          </div>
          <div className="button-row">
            <button
              className="primary-button"
              onClick={handleCreateUploadUrl}
              disabled={!proposal || pendingAction !== null}
            >
              {pendingAction === "create-upload" ? "Gerando..." : "Gerar upload"}
            </button>
            <button
              className="ghost-button"
              onClick={() => latestDocument && handleConfirmDocument(latestDocument.document_id)}
              disabled={!latestDocument || pendingAction !== null}
            >
              Confirmar ultimo envio
            </button>
          </div>
          {latestDocument ? (
            <div className="upload-preview">
              <span>Ultimo documento</span>
              <strong>{latestDocument.document_id}</strong>
              <a href={latestDocument.upload_url} target="_blank" rel="noreferrer">
                Abrir URL simulada
              </a>
            </div>
          ) : null}
        </article>

        <article className="card wide-card">
          <div className="card-header">
            <span className="step-index">04</span>
            <div>
              <h2>Painel consolidado</h2>
              <p>Status, cliente e documentos da proposta atual.</p>
            </div>
          </div>
          <div className="timeline">
            <div className="timeline-item">
              <span className="timeline-label">Proposta</span>
              <strong>{proposal ? formatStatus(proposal.status) : "Nao iniciada"}</strong>
            </div>
            <div className="timeline-item">
              <span className="timeline-label">Cliente</span>
              <strong>{proposal?.customer?.full_name ?? "Aguardando cadastro"}</strong>
            </div>
            <div className="timeline-item">
              <span className="timeline-label">Documentos</span>
              <strong>{documents.length > 0 ? `${documents.length} registrado(s)` : "Nenhum"}</strong>
            </div>
          </div>

          <div className="collection">
            {documents.length === 0 ? (
              <p className="empty-state">Ainda nao ha documentos registrados para esta proposta.</p>
            ) : (
              documents.map((document) => (
                <div className="collection-item" key={document.document_id}>
                  <div>
                    <strong>{document.document_type}</strong>
                    <span>{document.file_name}</span>
                  </div>
                  <div className="collection-actions">
                    <span className={`status-pill status-${document.status}`}>{document.status}</span>
                    {document.status !== "uploaded" ? (
                      <button
                        className="ghost-button"
                        onClick={() => handleConfirmDocument(document.document_id)}
                        disabled={pendingAction !== null}
                      >
                        {pendingAction === `confirm-${document.document_id}`
                          ? "Confirmando..."
                          : "Confirmar"}
                      </button>
                    ) : null}
                  </div>
                </div>
              ))
            )}
          </div>

          <div className="analysis-section">
            <div className="analysis-header">
              <h3>Resultados das analises</h3>
              <p>Persistidos no proposal service pelo workflow do MVP.</p>
            </div>
            {analysisResults.length === 0 ? (
              <p className="empty-state">As analises ainda nao foram executadas para esta proposta.</p>
            ) : (
              <div className="collection">
                {analysisResults.map((result) => (
                  <div className="collection-item" key={result.analysis_type}>
                    <div>
                      <strong>{analysisLabels[result.analysis_type] ?? result.analysis_type}</strong>
                      <span>{result.reason}</span>
                    </div>
                    <div className="collection-actions">
                      <span className={`status-pill status-${result.result}`}>{result.result}</span>
                      <span className="score-pill">score {result.score}</span>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="analysis-section">
            <div className="analysis-header">
              <h3>Timeline da proposta</h3>
              <p>Historico de status e comunicacoes registradas no fluxo.</p>
            </div>
            {timelineEvents.length === 0 ? (
              <p className="empty-state">Ainda nao ha eventos registrados para esta proposta.</p>
            ) : (
              <div className="timeline-stack">
                {timelineEvents.map((event) => (
                  <div className="timeline-row" key={event.id}>
                    <div className={`timeline-badge timeline-${event.kind}`}>{event.kind}</div>
                    <div className="timeline-content">
                      <strong>{event.title}</strong>
                      <span>{event.detail}</span>
                    </div>
                    <time>{new Date(event.timestamp).toLocaleString("pt-BR")}</time>
                  </div>
                ))}
              </div>
            )}
          </div>
        </article>
      </section>

      <footer className="footer-bar">
        <span>{message}</span>
        <span>API alvo: {process.env.NEXT_PUBLIC_BFF_URL ?? "http://localhost:8080"}</span>
      </footer>
    </main>
  );
}
