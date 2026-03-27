export type Customer = {
  customer_id: string;
  proposal_id: string;
  full_name: string;
  cpf: string;
  birth_date: string;
  email: string;
  phone: string;
  monthly_income: number;
  address?: string;
};

export type Document = {
  document_id: string;
  proposal_id: string;
  document_type: string;
  file_name: string;
  content_type: string;
  file_key: string;
  status: string;
  upload_url: string;
  storage_url?: string;
  uploaded_at?: string;
};

export type Proposal = {
  proposal_id: string;
  protocol: string;
  status: string;
  customer?: Customer;
  documents?: Document[];
  analysis_results?: AnalysisResult[];
  status_history?: StatusHistoryEntry[];
  notifications?: NotificationEntry[];
  created_at: string;
  updated_at: string;
};

export type AnalysisResult = {
  analysis_id?: string;
  proposal_id: string;
  analysis_type: string;
  result: string;
  provider: string;
  score: number;
  reason: string;
  created_at?: string;
};

export type StatusHistoryEntry = {
  status_history_id: string;
  proposal_id: string;
  status: string;
  source: string;
  created_at: string;
};

export type NotificationEntry = {
  notification_id: string;
  proposal_id: string;
  channel: string;
  template: string;
  recipient: string;
  message: string;
  status: string;
  trigger_status: string;
  sent_at: string;
  created_at: string;
};

type CustomerPayload = {
  full_name: string;
  cpf: string;
  birth_date: string;
  email: string;
  phone: string;
  monthly_income: number;
  address: string;
};

type DocumentPayload = {
  document_type: string;
  file_name: string;
  content_type: string;
};

const DEFAULT_BFF_URL = process.env.NEXT_PUBLIC_BFF_URL ?? "http://localhost:8080";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${DEFAULT_BFF_URL}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      message: "Falha ao comunicar com a API.",
    }));
    throw new Error(error.message ?? "Falha ao comunicar com a API.");
  }

  return response.json() as Promise<T>;
}

export async function createProposal() {
  return request<{ proposal_id: string; protocol: string; status: string }>("/api/v1/proposals", {
    method: "POST",
  });
}

export async function getProposal(proposalId: string) {
  return request<Proposal>(`/api/v1/proposals/${proposalId}`, {
    method: "GET",
  });
}

export async function upsertCustomer(proposalId: string, payload: CustomerPayload) {
  return request<{ status: string; message: string }>(`/api/v1/proposals/${proposalId}/customer`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function createDocumentUploadUrl(proposalId: string, payload: DocumentPayload) {
  return request<Document>(`/api/v1/proposals/${proposalId}/documents/upload-url`, {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function uploadDocumentContent(proposalId: string, documentId: string, file: File) {
  const formData = new FormData();
  formData.append("file", file);

  const response = await fetch(
    `${DEFAULT_BFF_URL}/api/v1/proposals/${proposalId}/documents/${documentId}/content`,
    {
      method: "POST",
      body: formData,
    },
  );

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      message: "Falha ao enviar documento.",
    }));
    throw new Error(error.message ?? "Falha ao enviar documento.");
  }

  return response.json() as Promise<Document>;
}

export async function confirmDocumentReceived(proposalId: string, documentId: string) {
  return request<Document>(`/api/v1/proposals/${proposalId}/documents/${documentId}/received`, {
    method: "POST",
  });
}
