import { useEffect, useState } from "react";
import { toast } from "sonner";
import { adminApi, requestsApi } from "@/lib/services";
import { apiError } from "@/lib/api";
import { formatDate, formatINR } from "@/lib/utils";
import type { Booking, Event, User, VenueRequest } from "@/types/schemas";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { StatusBadge } from "@/components/layout/StatusBadge";
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";

export default function AdminDashboard() {
  return (
    <div className="container py-10">
      <h1 className="mb-6 text-3xl font-bold">Admin dashboard</h1>
      <Tabs defaultValue="requests">
        <TabsList>
          <TabsTrigger value="requests">Claim requests</TabsTrigger>
          <TabsTrigger value="events">Events</TabsTrigger>
          <TabsTrigger value="bookings">Bookings</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
        </TabsList>
        <TabsContent value="requests"><RequestsTab /></TabsContent>
        <TabsContent value="events"><EventsTab /></TabsContent>
        <TabsContent value="bookings"><BookingsTab /></TabsContent>
        <TabsContent value="users"><UsersTab /></TabsContent>
      </Tabs>
    </div>
  );
}

function RequestsTab() {
  const [requests, setRequests] = useState<VenueRequest[]>([]);
  const [rejectTarget, setRejectTarget] = useState<VenueRequest | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);

  const load = () => {
    requestsApi.adminList().then(setRequests).catch((err) => toast.error(apiError(err)));
  };
  useEffect(load, []);

  const approve = async (id: string) => {
    setBusy(true);
    try {
      await requestsApi.approve(id);
      toast.success("Approved — venue claimed");
      load();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  const reject = async () => {
    if (!rejectTarget) return;
    setBusy(true);
    try {
      await requestsApi.reject(rejectTarget.id, reason || "Not approved");
      toast.success("Request rejected");
      setRejectTarget(null);
      setReason("");
      load();
    } catch (err) {
      toast.error(apiError(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Venue</TableHead>
            <TableHead>Organizer</TableHead>
            <TableHead>Document</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Submitted</TableHead>
            <TableHead className="text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {requests.map((r) => (
            <TableRow key={r.id}>
              <TableCell className="font-mono text-xs">{r.venueId.slice(0, 8)}…</TableCell>
              <TableCell className="font-mono text-xs">{r.organizerId.slice(0, 8)}…</TableCell>
              <TableCell>{r.documentMock}</TableCell>
              <TableCell><StatusBadge status={r.status} /></TableCell>
              <TableCell>{formatDate(r.createdAt)}</TableCell>
              <TableCell className="text-right">
                {r.status === "pending" ? (
                  <div className="flex justify-end gap-2">
                    <Button size="sm" disabled={busy} onClick={() => approve(r.id)}>Approve</Button>
                    <Button size="sm" variant="destructive" disabled={busy} onClick={() => setRejectTarget(r)}>
                      Reject
                    </Button>
                  </div>
                ) : (
                  <span className="text-xs text-muted-foreground">{r.rejectionReason ?? "—"}</span>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      <Dialog open={!!rejectTarget} onOpenChange={(o) => !o && setRejectTarget(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>Reject claim</DialogTitle></DialogHeader>
          <div className="space-y-2">
            <Label>Reason</Label>
            <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Document invalid" />
          </div>
          <DialogFooter>
            <Button variant="destructive" disabled={busy} onClick={reject}>Reject request</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function EventsTab() {
  const [events, setEvents] = useState<Event[]>([]);
  useEffect(() => {
    adminApi.events().then(setEvents).catch((err) => toast.error(apiError(err)));
  }, []);
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Title</TableHead>
          <TableHead>Date</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Organizer</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {events.map((e) => (
          <TableRow key={e.id}>
            <TableCell className="font-medium">{e.title}</TableCell>
            <TableCell>{formatDate(e.eventDate)}</TableCell>
            <TableCell><StatusBadge status={e.status} /></TableCell>
            <TableCell className="font-mono text-xs">{e.organizerId.slice(0, 8)}…</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function BookingsTab() {
  const [bookings, setBookings] = useState<Booking[]>([]);
  useEffect(() => {
    adminApi.bookings().then(setBookings).catch((err) => toast.error(apiError(err)));
  }, []);
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Booking</TableHead>
          <TableHead>User</TableHead>
          <TableHead>Seats</TableHead>
          <TableHead>Amount</TableHead>
          <TableHead>Status</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {bookings.map((b) => (
          <TableRow key={b.id}>
            <TableCell className="font-mono text-xs">{b.id.slice(0, 8)}…</TableCell>
            <TableCell className="font-mono text-xs">{b.userId.slice(0, 8)}…</TableCell>
            <TableCell>{b.seatIds.length}</TableCell>
            <TableCell>{formatINR(b.totalAmount)}</TableCell>
            <TableCell><StatusBadge status={b.status} /></TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function UsersTab() {
  const [users, setUsers] = useState<User[]>([]);
  useEffect(() => {
    adminApi.users().then(setUsers).catch((err) => toast.error(apiError(err)));
  }, []);
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Email</TableHead>
          <TableHead>Role</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Joined</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {users.map((u) => (
          <TableRow key={u.id}>
            <TableCell>{u.email}</TableCell>
            <TableCell><StatusBadge status={u.role} /></TableCell>
            <TableCell><StatusBadge status={u.status} /></TableCell>
            <TableCell>{formatDate(u.createdAt)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
