import { redirect } from "next/navigation";

import { getAuthConfig } from "@/lib/auth/config";
import { getServerSession } from "@/lib/auth/session";

export default async function LoginPage() {
  const session = await getServerSession();
  if (session) {
    redirect("/");
  }

  const authConfig = getAuthConfig();

  return (
    <main className="auth-shell">
      <section className="auth-panel auth-panel-copy">
        <p className="eyebrow">Acesso</p>
        <h1>Entrar para operar a jornada do Credit Flow.</h1>
        <p className="lead">
          O ambiente atual usa autenticacao mock com sessao em cookie. A estrutura de
          configuracao ja aceita migracao futura para OIDC sem trocar a experiencia principal.
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
        </div>
      </section>

      <section className="auth-panel auth-panel-form">
        <div className="card-header">
          <span className="step-index">ID</span>
          <div>
            <h2>Login operacional</h2>
            <p>Use o modo mock para navegar no MVP e preservar a porta de migracao para OIDC.</p>
          </div>
        </div>

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
      </section>
    </main>
  );
}
