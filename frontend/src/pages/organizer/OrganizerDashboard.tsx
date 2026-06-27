import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { toast } from "sonner";
import { CreditCard } from "lucide-react";
import { authApi, eventsApi, requestsApi, venuesApi } from "@/lib/services";
import { apiError } from "@/lib/api";
import { formatDate } from "@/lib/utils";
import { useAuth } from "@/store/auth";
import type { Event, Venue, VenueRequest } from "@/types/schemas";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusBadge } from "@/components/layout/StatusBadge";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";

export default function OrganizerDashboard() {
  const { user, refreshUser } = useAuth();
  const [venues, setVenues] = useState<Venue[]>([]);
  const [requests, setRequests] = useState<VenueRequest[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [activating, setActivating] = useState(false);

  const isPending = user?.status === "pending_payment";

  const loadAll = async () => {
    try {
      const [vns, reqs, evs] = await Promise.all([
        venuesApi.list("unclaimed"),
        requestsApi.mine().catch(() => []),
        eventsApi.list().catch(() => []),
      ]);
      setVenues(vns);
      setRequests(reqs);
      setEvents(evs.filter((e) => e.organizerId === user?.id));
    } catch (err) {
      toast.error(apiError(err));
    }
  };

  useEffect(() => {
    if (!isPending) loadAll();
  }, [isPending]);

  const activate = async () => {
    setActivating(true);
    try {
      const pair = await authApi.activate({ method: "mock_card", card: "4242" });
      await refreshUser();
      toast.success("Account activated! You can now claim venues.");
      void pair;
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setActivating(false);
    }
  };

  if (isPending) {
    return (
      <div className="container flex justify-center py-16">
        <Card className="w-full max-w-md border-amber-500/40">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <CreditCard className="h-5 w-5" /> Activate your organizer account
            </CardTitle>
            <CardDescription>
              Pay the one-time platform registration fee to start claiming venues and running events.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button className="w-full" disabled={activating} onClick={activate}>
              {activating ? "Processing payment…" : "Pay mock fee & activate"}
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="container py-10">
      <h1 className="mb-6 text-3xl font-bold">Organizer dashboard</h1>
      <Tabs defaultValue="events">
        <TabsList>
          <TabsTrigger value="events">My events</TabsTrigger>
          <TabsTrigger value="venues">Claim a venue</TabsTrigger>
          <TabsTrigger value="requests">My requests</TabsTrigger>
        </TabsList>

        <TabsContent value="events">
          <EventsTab events={events} venues={venues} onChanged={loadAll} />
        </TabsContent>
        <TabsContent value="venues">
          <VenuesTab venues={venues} onClaimed={loadAll} />
        </TabsContent>
        <TabsContent value="requests">
          <RequestsTab requests={requests} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ---- My events tab ----
function EventsTab({
  events, venues, onChanged,
}: { events: Event[]; venues: Venue[]; onChanged: () => void }) {
  const [open, setOpen] = useState(false);
  // Venues this organizer owns (claimed) — fetched separately since the
  // unclaimed list won't include them.
  const [myVenues, setMyVenues] = useState<Venue[]>([]);

  useEffect(() => {
    venuesApi.list("claimed").then(setMyVenues).catch(() => setMyVenues([]));
  }, [events.length]);
  void venues;

  return (
    <div>
      <div className="mb-4 flex justify-end">
        <Button onClick={() => setOpen(true)} disabled={myVenues.length === 0}>
          Create event
        </Button>
      </div>
      {myVenues.length === 0 && (
        <p className="mb-4 text-sm text-muted-foreground">
          Claim and get a venue approved before creating an event.
        </p>
      )}
      {events.length === 0 ? (
        <p className="text-muted-foreground">No events yet.</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Title</TableHead>
              <TableHead>Date</TableHead>
              <TableHead>Status</TableHead>
              <TableHead></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {events.map((ev) => (
              <TableRow key={ev.id}>
                <TableCell className="font-medium">{ev.title}</TableCell>
                <TableCell>{formatDate(ev.eventDate)}</TableCell>
                <TableCell><StatusBadge status={ev.status} /></TableCell>
                <TableCell className="text-right">
                  <Link to={`/organizer/events/${ev.id}`}>
                    <Button variant="outline" size="sm">Manage</Button>
                  </Link>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <CreateEventDialog
        open={open}
        onOpenChange={setOpen}
        venues={myVenues}
        onCreated={onChanged}
      />
    </div>
  );
}

function CreateEventDialog({
  open, onOpenChange, venues, onCreated,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  venues: Venue[];
  onCreated: () => void;
}) {
  const [venueId, setVenueId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [eventDate, setEventDate] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    setBusy(true);
    try {
      await eventsApi.create({ venueId, title, description, eventDate });
      toast.success("Event created as draft");
      onOpenChange(false);
      setTitle(""); setDescription(""); setEventDate(""); setVenueId("");
      onCreated();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create event</DialogTitle>
          <DialogDescription>One active event per venue at a time.</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>Venue</Label>
            <Select value={venueId} onValueChange={setVenueId}>
              <SelectTrigger><SelectValue placeholder="Pick a claimed venue" /></SelectTrigger>
              <SelectContent>
                {venues.map((v) => (
                  <SelectItem key={v.id} value={v.id}>{v.name} — {v.city}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label>Title</Label>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>Description</Label>
            <Textarea value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label>Date & time</Label>
            <Input type="datetime-local" value={eventDate} onChange={(e) => setEventDate(e.target.value)} />
          </div>
        </div>
        <DialogFooter>
          <Button disabled={busy || !venueId || !title || !eventDate} onClick={submit}>
            {busy ? "Creating…" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ---- Claim a venue tab ----
function VenuesTab({ venues, onClaimed }: { venues: Venue[]; onClaimed: () => void }) {
  const [target, setTarget] = useState<Venue | null>(null);
  const [doc, setDoc] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!target) return;
    setBusy(true);
    try {
      await requestsApi.submit(target.id, doc || "ownership-proof.pdf");
      toast.success("Claim request submitted for review");
      setTarget(null);
      setDoc("");
      onClaimed();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {venues.map((v) => (
          <Card key={v.id}>
            <CardHeader>
              <CardTitle className="text-base">{v.name}</CardTitle>
              <CardDescription>{v.address}, {v.city}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <p className="text-sm text-muted-foreground">Capacity {v.capacity.toLocaleString()}</p>
              <Button size="sm" onClick={() => setTarget(v)}>Claim</Button>
            </CardContent>
          </Card>
        ))}
      </div>

      <Dialog open={!!target} onOpenChange={(o) => !o && setTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Claim {target?.name}</DialogTitle>
            <DialogDescription>Attach a mock supporting document for the admin to review.</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Supporting document (mock)</Label>
            <Input value={doc} onChange={(e) => setDoc(e.target.value)} placeholder="lease-agreement.pdf" />
          </div>
          <DialogFooter>
            <Button disabled={busy} onClick={submit}>
              {busy ? "Submitting…" : "Submit claim"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---- My requests tab ----
function RequestsTab({ requests }: { requests: VenueRequest[] }) {
  if (requests.length === 0) {
    return <p className="text-muted-foreground">No claim requests yet.</p>;
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Venue ID</TableHead>
          <TableHead>Document</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Reason</TableHead>
          <TableHead>Submitted</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {requests.map((r) => (
          <TableRow key={r.id}>
            <TableCell className="font-mono text-xs">{r.venueId.slice(0, 8)}…</TableCell>
            <TableCell>{r.documentMock}</TableCell>
            <TableCell><StatusBadge status={r.status} /></TableCell>
            <TableCell className="text-muted-foreground">{r.rejectionReason ?? "—"}</TableCell>
            <TableCell>{formatDate(r.createdAt)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
