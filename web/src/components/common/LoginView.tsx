import { useAuthStore } from "@/stores";
import { BookOpen, LogIn } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";

export function LoginView() {
  const { login, loading, error, clearError, isLoginModalOpen, setLoginModalOpen } = useAuthStore();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const { t } = useTranslation();

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    clearError();
    await login(email, password)
      .then(() => {
        setLoginModalOpen(false);
      })
      .catch(() => undefined);
  }

  return (
    <dialog className={`modal ${isLoginModalOpen ? "modal-open" : ""}`}>
      <div className="modal-box">
        <button 
          onClick={() => setLoginModalOpen(false)}
          className="btn btn-sm btn-circle btn-ghost absolute right-2 top-2"
        >
          ✕
        </button>
        <div className="flex flex-col items-center gap-4 mb-8 mt-2 text-center">
          <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
            <BookOpen size={28} className="text-primary" />
          </div>
          <div>
            <h3 className="text-2xl font-bold">NovelHub</h3>
            <p className="text-base-content/60 font-medium text-sm mt-1">{t("auth.login_to_account")}</p>
          </div>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="form-control w-full">
            <label className="label">
              <span className="label-text font-semibold">{t("auth.email")}</span>
            </label>
            <input 
              value={email} 
              onChange={(event) => setEmail(event.target.value)} 
              type="email" 
              placeholder={"account@example.com"}
              autoComplete="email" 
              className="input input-bordered w-full focus:input-primary"
            />
          </div>
          <div className="form-control w-full">
            <label className="label">
              <span className="label-text font-semibold">{t("auth.password")}</span>
            </label>
            <input
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              type="password"
              placeholder={"********"}
              autoComplete="current-password"
              className="input input-bordered w-full focus:input-primary"
            />
          </div>

          {error && (
            <div className="alert alert-error mt-2 py-2 text-sm rounded-lg">
              <span>{error}</span>
            </div>
          )}

          <button 
            className="btn btn-primary mt-4 w-full" 
            disabled={loading}
          >
            {loading ? <span className="loading loading-spinner"></span> : <LogIn size={20} />}
            {t("auth.sign_in")}
          </button>
        </form>
      </div>
      <form method="dialog" className="modal-backdrop">
        <button onClick={() => setLoginModalOpen(false)}>close</button>
      </form>
    </dialog>
  );
}
