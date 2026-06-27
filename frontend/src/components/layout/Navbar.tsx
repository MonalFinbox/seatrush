import { Link, useNavigate } from "react-router-dom";
import { Ticket, LogOut } from "lucide-react";
import { useAuth } from "@/store/auth";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

export function Navbar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const onLogout = () => {
    logout();
    navigate("/login");
  };

  return (
    <header className="sticky top-0 z-40 w-full border-b bg-background/95 backdrop-blur">
      <div className="container flex h-14 items-center justify-between">
        <Link to="/" className="flex items-center gap-2 font-bold">
          <Ticket className="h-5 w-5" />
          SeatRush
        </Link>

        <nav className="flex items-center gap-2 text-sm">
          <Link to="/">
            <Button variant="ghost" size="sm">Events</Button>
          </Link>

          {user?.role === "attendee" && (
            <Link to="/bookings">
              <Button variant="ghost" size="sm">My Bookings</Button>
            </Link>
          )}
          {user?.role === "organizer" && (
            <Link to="/organizer">
              <Button variant="ghost" size="sm">Organizer</Button>
            </Link>
          )}
          {user?.role === "admin" && (
            <Link to="/admin">
              <Button variant="ghost" size="sm">Admin</Button>
            </Link>
          )}

          {user ? (
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="hidden sm:inline-flex">
                {user.email} · {user.role}
              </Badge>
              <Button variant="outline" size="sm" onClick={onLogout}>
                <LogOut className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <>
              <Link to="/login">
                <Button variant="ghost" size="sm">Login</Button>
              </Link>
              <Link to="/register">
                <Button size="sm">Sign up</Button>
              </Link>
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
