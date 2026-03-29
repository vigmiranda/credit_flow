import { redirect } from "next/navigation";

import { getAuthConfig } from "@/lib/auth/config";
import { getServerSession } from "@/lib/auth/session";

export default async function LoginPage() {
  const session = await getServerSession();
  if (session) {
    redirect("/");
  }

  const authConfig = getAuthConfig();
  const oidcReady = Boolean(
    authConfig.issuerUrl &&
      authConfig.clientId &&
      authConfig.redirectUri &&
      authConfig.authorizeUrl &&
      authConfig.tokenUrl,
  );

  return (
    <main className="auth-shell">
      <section className="auth-panel auth-panel-copy">
        <p className="eyebrow">Acesso</p>
        <h1>Entrar para operar a jornada do Credit Flow.</h1>
        <p className="lead">
          O ambiente atual suporta sessao mock e ja aceita redirecionamento para um provedor
          OIDC com callback dedicado no front.
        </p>
        <div className="auth-hints">
          <div>
            <span>Modo atual</span>
            <strong>{authConfig.mode}</strong>
          </div>
          <div>
            <span>Issuer configurado</span>
            <strong>{authConfig.issuerUrl || "nao definido"}</strong>
          </div>
          <div>
            <span>Redirect previsto</span>
            <strong>{authConfig.redirectUri}</strong>
          </div>
          <div>
            <span>Discovery URL</span>
            <strong>{authConfig.discoveryUrl || "nao definido"}</strong>
          </div>
          <div>
            <span>Authorize URL</span>
            <strong>{authConfig.authorizeUrl || "nao definido"}</strong>
          </div>
          <div>
            <span>Token URL</span>
            <strong>{authConfig.tokenUrl || "nao definido"}</strong>
          </div>
        </div>
      </section>

      <section className="auth-panel auth-panel-form">
        <div className="card-header">
          <span className="step-index">ID</span>
          <div>
            <h2>Login operacional</h2>
            <p>Use login mock no ambiente local ou inicie o fluxo OIDC quando o issuer estiver configurado.</p>
          </div>
        </div>

        {authConfig.mode === "mock" ? (
          <form action="/api/auth/mock-login" method="post" className="form-grid">
            <label>
              Nome
              <input name="name" defaultValue="Maria Operadora" placeholder="Maria Operadora" />
            </label>
            <label>
              E-mail
              <input
                name="email"
                type="email"
                defaultValue="maria.operadora@creditflow.local"
                placeholder="maria.operadora@creditflow.local"
              />
            </label>
            <label>
              Perfil
              <select name="role" defaultValue="analyst">
                <option value="analyst">analyst</option>
                <option value="supervisor">supervisor</option>
                <option value="admin">admin</option>
              </select>
            </label>
            <label>
              User ID
              <input name="user_id" defaultValue="usr_mock_001" placeholder="usr_mock_001" />
            </label>
            <div className="auth-actions">
              <button className="primary-button" type="submit">
                Entrar no MVP
              </button>
            </div>
          </form>
        ) : (
          <div className="form-grid">
            <div className="full-width">
              <p className="lead">
                O front valida `state`, troca `code` por token e monta a sessao local a partir de `userinfo` ou `id_token`.
              </p>
            </div>
            <div className="auth-actions">
              <a
                className="primary-button"
                href={oidcReady ? "/api/auth/oidc-login" : "#"}
                aria-disabled={!oidcReady}
              >
                Iniciar login OIDC
              </a>
            </div>
            {!oidcReady ? (
              <p className="empty-state">
                Configure `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_REDIRECT_URI` e o endpoint de token para habilitar o fluxo.
              </p>
            ) : null}
          </div>
        )}
      </section>
    </main>
  );
}
