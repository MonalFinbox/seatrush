import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { ShieldAlert } from "lucide-react";
import { authApi } from "@/lib/services";
import { apiError } from "@/lib/api";
import { adminLoginFormSchema } from "@/types/schemas";
import { useAuth } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

/**
 * This route is intentionally unlinked from the rest of the UI — admins reach
 * it directly. It requires the static admin access key on top of credentials.
 */
export default function AdminLogin() {
  const { setSession } = useAuth();
  const navigate = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [adminAccessKey, setKey] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    const parsed = adminLoginFormSchema.safeParse({ email, password, adminAccessKey });
    if (!parsed.success) {
      toast.error(parsed.error.issues[0].message);
      return;
    }
    setBusy(true);
    try {
      const pair = await authApi.adminLogin(parsed.data);
      setSession(pair);
      toast.success("Admin session started");
      navigate("/admin");
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="container flex justify-center py-16">
      <Card className="w-full max-w-sm border-amber-500/40">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <ShieldAlert className="h-5 w-5 text-amber-500" /> Admin access
          </CardTitle>
          <CardDescription>Restricted. Seeded admin accounts only.</CardDescription>
        </CardHeader>
        <form onSubmit={submit}>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input id="password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="key">Admin access key</Label>
              <Input id="key" type="password" value={adminAccessKey} onChange={(e) => setKey(e.target.value)} />
            </div>
          </CardContent>
          <CardFooter>
            <Button type="submit" className="w-full" disabled={busy}>
              {busy ? "Authenticating…" : "Enter dashboard"}
            </Button>
          </CardFooter>
        </form>
      </Card>
    </div>
  );
}
