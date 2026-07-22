import { settingsService } from "@/services";
import { BookOpen, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { PasswordStrength } from "@/components/common";

export function RegisterPage() {
  const navigate = useNavigate();
  const [form, setForm] = useState({ email: "", password: "", full_name: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [registrationEnabled, setRegistrationEnabled] = useState(true);

  useEffect(() => {
    settingsService.getPublic().then((res) => {
      if (res.data && !res.data.registration_enabled) {
        setRegistrationEnabled(false);
      }
    });
  }, []);

  const handleSubmit = async (e: React.SyntheticEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await settingsService.register({
        email: form.email,
        password: form.password,
        full_name: form.full_name || undefined,
      });
      if (res.status) {
        navigate("/", { replace: true });
      } else {
        setError(res.message || "Registration failed");
      }
    } catch {
      setError("An unexpected error occurred");
    } finally {
      setLoading(false);
    }
  };

  if (!registrationEnabled) {
    return (
      <div className="min-h-screen bg-base-200 flex items-center justify-center p-4">
        <div className="card w-full max-w-md bg-base-100 shadow-xl text-center">
          <div className="card-body items-center gap-4">
            <BookOpen size={40} className="text-base-content/30" />
            <h2 className="text-xl font-bold">Registration Disabled</h2>
            <p className="text-sm text-base-content/60">
              Public registration is currently disabled by the administrator.
            </p>
            <Link to="/" className="btn btn-primary btn-sm">Go Home</Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-200 flex items-center justify-center p-4">
      <div className="card w-full max-w-md bg-base-100 shadow-xl">
        <div className="card-body">
          <div className="flex flex-col items-center gap-2 mb-4">
            <div className="w-14 h-14 rounded-2xl bg-primary/10 flex items-center justify-center">
              <BookOpen size={28} className="text-primary" />
            </div>
            <h2 className="text-2xl font-bold">Create Account</h2>
            <p className="text-sm text-base-content/60">
              Register to access the library features.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <div className="form-control">
              <label className="label"><span className="label-text font-semibold">Email</span></label>
              <input
                type="email"
                placeholder="account@example.com"
                className="input input-bordered w-full"
                value={form.email}
                onChange={(e) => setForm({ ...form, email: e.target.value })}
                required
                autoComplete="email"
              />
            </div>
            <div className="form-control">
              <label className="label"><span className="label-text font-semibold">Full Name</span></label>
              <input
                type="text"
                placeholder="(optional)"
                className="input input-bordered w-full"
                value={form.full_name}
                onChange={(e) => setForm({ ...form, full_name: e.target.value })}
              />
            </div>
            <div className="form-control">
              <label className="label"><span className="label-text font-semibold">Password</span></label>
              <input
                type="password"
                placeholder="Minimum 8 characters"
                className="input input-bordered w-full"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                required
                minLength={8}
                autoComplete="new-password"
              />
              {form.password.length > 0 && <PasswordStrength password={form.password} />}
            </div>

            {error && (
              <div className="alert alert-error py-2 text-sm rounded-lg">{error}</div>
            )}

            <button className="btn btn-primary mt-2" disabled={loading}>
              {loading ? <Loader2 className="animate-spin" size={20} /> : null}
              Register
            </button>
          </form>

          <div className="text-center mt-2">
            <Link to="/" className="text-sm link link-hover">Back to library</Link>
          </div>
        </div>
      </div>
    </div>
  );
}
