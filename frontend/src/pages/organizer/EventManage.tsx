import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { toast } from "sonner";
import { eventsApi, seatsApi, type SeatInput } from "@/lib/services";
import { apiError } from "@/lib/api";
import { formatDate, formatINR } from "@/lib/utils";
import type { Event, Seat } from "@/types/schemas";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/layout/StatusBadge";
import { SeatMap } from "@/components/SeatMap";

export default function EventManage() {
  const { eventId } = useParams<{ eventId: string }>();
  const [event, setEvent] = useState<Event | null>(null);
  const [seats, setSeats] = useState<Seat[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  // Seat-map builder state.
  const [section, setSection] = useState("A");
  const [rows, setRows] = useState(5);
  const [perRow, setPerRow] = useState(10);
  const [price, setPrice] = useState(500);
  const [draftSeats, setDraftSeats] = useState<SeatInput[]>([]);

  const load = useCallback(async () => {
    if (!eventId) return;
    try {
      const [ev, map] = await Promise.all([eventsApi.get(eventId), seatsApi.map(eventId)]);
      setEvent(ev);
      setSeats(map);
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setLoading(false);
    }
  }, [eventId]);

  useEffect(() => {
    load();
  }, [load]);

  const addBlock = () => {
    const block: SeatInput[] = [];
    for (let r = 1; r <= rows; r++) {
      for (let n = 1; n <= perRow; n++) {
        block.push({ section, row: String(r), number: String(n), price });
      }
    }
    setDraftSeats((prev) => [...prev, ...block]);
    toast.success(`Added section ${section}: ${block.length} seats`);
  };

  const saveSeats = async () => {
    if (!eventId || draftSeats.length === 0) return;
    setBusy(true);
    try {
      const res = await seatsApi.create(eventId, draftSeats);
      toast.success(`Seat map saved (${res.created} seats)`);
      setDraftSeats([]);
      load();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  const publish = async () => {
    if (!eventId) return;
    setBusy(true);
    try {
      await eventsApi.publish(eventId);
      toast.success("Event published");
      load();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  const cancel = async () => {
    if (!eventId) return;
    setBusy(true);
    try {
      await eventsApi.cancel(eventId);
      toast.success("Event cancelled");
      load();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  if (loading) return <div className="container py-10"><Skeleton className="h-72 w-full" /></div>;
  if (!event) return <div className="container py-10">Event not found.</div>;

  const hasSeats = seats.length > 0;
  const draftTotal = draftSeats.length;

  return (
    <div className="container grid gap-6 py-10 lg:grid-cols-3">
      <div className="lg:col-span-2 space-y-4">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{event.title}</h1>
          <StatusBadge status={event.status} />
        </div>
        <p className="text-sm text-muted-foreground">{formatDate(event.eventDate)}</p>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Seat map</CardTitle>
            <CardDescription>
              {hasSeats ? `${seats.length} seats defined` : "Define the seat map before publishing."}
            </CardDescription>
          </CardHeader>
          <CardContent>
            {hasSeats ? (
              <SeatMap seats={seats} selected={new Set()} onToggle={() => {}} interactive={false} />
            ) : (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                  <div className="space-y-1">
                    <Label>Section</Label>
                    <Input value={section} onChange={(e) => setSection(e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <Label>Rows</Label>
                    <Input type="number" min={1} value={rows} onChange={(e) => setRows(+e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <Label>Seats/row</Label>
                    <Input type="number" min={1} value={perRow} onChange={(e) => setPerRow(+e.target.value)} />
                  </div>
                  <div className="space-y-1">
                    <Label>Price (₹)</Label>
                    <Input type="number" min={0} value={price} onChange={(e) => setPrice(+e.target.value)} />
                  </div>
                </div>
                <Button variant="outline" onClick={addBlock}>Add section block</Button>
                {draftTotal > 0 && (
                  <div className="flex items-center justify-between border-t pt-3">
                    <span className="text-sm text-muted-foreground">{draftTotal} seats staged</span>
                    <Button disabled={busy} onClick={saveSeats}>
                      {busy ? "Saving…" : "Save seat map"}
                    </Button>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      <div className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Lifecycle</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {event.status === "draft" && (
              <Button className="w-full" disabled={busy || !hasSeats} onClick={publish}>
                Publish event
              </Button>
            )}
            {!hasSeats && event.status === "draft" && (
              <p className="text-xs text-muted-foreground">Add a seat map to enable publishing.</p>
            )}
            {event.status !== "cancelled" && event.status !== "completed" && (
              <Button variant="destructive" className="w-full" disabled={busy} onClick={cancel}>
                Cancel event
              </Button>
            )}
            <p className="text-xs text-muted-foreground">
              Cancelling frees the venue for a new event.
            </p>
          </CardContent>
        </Card>

        {hasSeats && (
          <Card>
            <CardContent className="pt-6 text-sm text-muted-foreground">
              Cheapest seat: {formatINR(Math.min(...seats.map((s) => s.price)))}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  );
}
