import { Navigate, Outlet } from "react-router-dom";
import { useAuth } from "@/store/auth";

/** Gate routes that require a logged-in user of one of the allowed roles. */
export function ProtectedRoute({ roles }: { roles?: Array<"admin" | "organizer" | "attendee"> }) {
  const { user, loading } = useAuth();

  if (loading) {
    return <div className="p-10 text-center text-muted-foreground">Loading…</div>;
  }
  if (!user) {
    return <Navigate to="/login" replace />;
  }
  if (roles && !roles.includes(user.role)) {
    return <Navigate to="/" replace />;
  }
  return <Outlet />;
}
