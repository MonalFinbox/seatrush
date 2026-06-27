import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { CalendarDays, MapPin, Wifi, WifiOff } from "lucide-react";
import { eventsApi, seatsApi, venuesApi, holdsApi, bookingsApi } from "@/lib/services";
import { apiError } from "@/lib/api";
import { formatDate, formatINR } from "@/lib/utils";
import { useEventSocket } from "@/hooks/useEventSocket";
import { useAuth } from "@/store/auth";
import type { Event, HoldResponse, Seat, Venue, WsEvent } from "@/types/schemas";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/layout/StatusBadge";
import { SeatMap } from "@/components/SeatMap";

export default function EventDetail() {
  const { eventId } = useParams<{ eventId: string }>();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [event, setEvent] = useState<Event | null>(null);
  const [venue, setVenue] = useState<Venue | null>(null);
  const [seats, setSeats] = useState<Seat[]>([]);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [hold, setHold] = useState<HoldResponse | null>(null);
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [busy, setBusy] = useState(false);

  const interactive = user?.role === "attendee" && event?.status === "published" && !hold;

  const load = useCallback(async () => {
    if (!eventId) return;
    try {
      const ev = await eventsApi.get(eventId);
      setEvent(ev);
      const [map, vn] = await Promise.all([
        seatsApi.map(eventId),
        venuesApi.get(ev.venueId).catch(() => null),
      ]);
      setSeats(map);
      setVenue(vn);
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setLoading(false);
    }
  }, [eventId]);

  useEffect(() => {
    load();
  }, [load]);

  // Live seat updates.
  const onWs = useCallback((msg: WsEvent) => {
    const next: Seat["status"] =
      msg.type === "seat.held" ? "held" : msg.type === "seat.booked" ? "booked" : "available";
    setSeats((prev) =>
      prev.map((s) => (s.id === msg.seatId ? { ...s, status: next } : s))
    );
  }, []);
  const { connected } = useEventSocket(eventId, onWs);

  // Hold countdown.
  useEffect(() => {
    if (!hold) return;
    const tick = () => {
      const left = Math.max(0, Math.floor((new Date(hold.expiresAt).getTime() - Date.now()) / 1000));
      setSecondsLeft(left);
      if (left <= 0) {
        setHold(null);
        toast.info("Your hold expired");
      }
    };
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [hold]);

  const toggle = (seat: Seat) => {
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(seat.id) ? next.delete(seat.id) : next.add(seat.id);
      return next;
    });
  };

  const selectedSeats = useMemo(
    () => seats.filter((s) => selected.has(s.id)),
    [seats, selected]
  );
  const selectedTotal = selectedSeats.reduce((sum, s) => sum + s.price, 0);

  const placeHold = async () => {
    if (!eventId || selected.size === 0) return;
    setBusy(true);
    try {
      const h = await holdsApi.create(eventId, [...selected]);
      setHold(h);
      setSelected(new Set());
      toast.success("Seats held — confirm within the timer");
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  const releaseHold = async () => {
    if (!hold) return;
    setBusy(true);
    try {
      await holdsApi.release(hold.holdId);
      setHold(null);
      toast.success("Hold released");
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  const book = async () => {
    if (!hold) return;
    setBusy(true);
    try {
      await bookingsApi.create(hold.holdId, { method: "mock_card", card: "4242" });
      setHold(null);
      toast.success("Booking confirmed!");
      navigate("/bookings");
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  if (loading) {
    return (
      <div className="container py-10">
        <Skeleton className="mb-4 h-10 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    );
  }
  if (!event) return <div className="container py-10">Event not found.</div>;

  const heldSeats = seats.filter((s) => hold?.seatIds.includes(s.id));
  const heldTotal = heldSeats.reduce((sum, s) => sum + s.price, 0);

  return (
    <div className="container grid gap-6 py-10 lg:grid-cols-3">
      {/* Seat map */}
      <div className="lg:col-span-2">
        <div className="mb-4 flex items-start justify-between gap-2">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{event.title}</h1>
              <StatusBadge status={event.status} />
            </div>
            <div className="mt-1 flex flex-wrap gap-4 text-sm text-muted-foreground">
              <span className="flex items-center gap-1.5">
                <CalendarDays className="h-4 w-4" /> {formatDate(event.eventDate)}
              </span>
              {venue && (
                <span className="flex items-center gap-1.5">
                  <MapPin className="h-4 w-4" /> {venue.name}, {venue.city}
                </span>
              )}
            </div>
          </div>
          <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
            {connected ? <Wifi className="h-4 w-4 text-emerald-500" /> : <WifiOff className="h-4 w-4" />}
            {connected ? "live" : "offline"}
          </span>
        </div>

        <Card>
          <CardContent className="pt-6">
            <SeatMap seats={seats} selected={selected} onToggle={toggle} interactive={!!interactive} />
          </CardContent>
        </Card>
      </div>

      {/* Side panel */}
      <div className="space-y-4">
        {!user && (
          <Card>
            <CardContent className="pt-6 text-sm text-muted-foreground">
              <Button className="w-full" onClick={() => navigate("/login")}>
                Log in to book
              </Button>
            </CardContent>
          </Card>
        )}

        {user?.role === "attendee" && !hold && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Selected seats</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {selectedSeats.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {event.status === "published"
                    ? "Tap available seats to select."
                    : "This event isn't open for booking."}
                </p>
              ) : (
                <>
                  <ul className="space-y-1 text-sm">
                    {selectedSeats.map((s) => (
                      <li key={s.id} className="flex justify-between">
                        <span>{s.section}{s.row}-{s.number}</span>
                        <span>{formatINR(s.price)}</span>
                      </li>
                    ))}
                  </ul>
                  <div className="flex justify-between border-t pt-2 font-semibold">
                    <span>Total</span>
                    <span>{formatINR(selectedTotal)}</span>
                  </div>
                  <Button className="w-full" disabled={busy} onClick={placeHold}>
                    {busy ? "Holding…" : `Hold ${selectedSeats.length} seat(s)`}
                  </Button>
                </>
              )}
            </CardContent>
          </Card>
        )}

        {hold && (
          <Card className="border-amber-500/50">
            <CardHeader>
              <CardTitle className="text-base flex items-center justify-between">
                Your hold
                <span className="font-mono text-amber-400">
                  {Math.floor(secondsLeft / 60)}:{String(secondsLeft % 60).padStart(2, "0")}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <ul className="space-y-1 text-sm">
                {heldSeats.map((s) => (
                  <li key={s.id} className="flex justify-between">
                    <span>{s.section}{s.row}-{s.number}</span>
                    <span>{formatINR(s.price)}</span>
                  </li>
                ))}
              </ul>
              <div className="flex justify-between border-t pt-2 font-semibold">
                <span>Total</span>
                <span>{formatINR(heldTotal)}</span>
              </div>
              <Button className="w-full" disabled={busy} onClick={book}>
                {busy ? "Processing…" : "Pay & confirm (mock)"}
              </Button>
              <Button variant="outline" className="w-full" disabled={busy} onClick={releaseHold}>
                Release hold
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
