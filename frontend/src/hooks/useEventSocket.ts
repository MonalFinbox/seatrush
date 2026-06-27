import { useEffect, useRef, useState } from "react";
import { wsEventSchema, type WsEvent } from "@/types/schemas";

const WS_URL = import.meta.env.VITE_WS_URL ?? "ws://localhost:8080/ws/v1";

/**
 * useEventSocket subscribes to live seat events for one event and invokes
 * onEvent for each message. It reconnects automatically if the socket drops.
 */
export function useEventSocket(
  eventId: string | undefined,
  onEvent: (e: WsEvent) => void
) {
  const [connected, setConnected] = useState(false);
  // Keep the latest callback without re-opening the socket on every render.
  const cbRef = useRef(onEvent);
  cbRef.current = onEvent;

  useEffect(() => {
    if (!eventId) return;
    let socket: WebSocket | null = null;
    let retry: ReturnType<typeof setTimeout> | null = null;
    let closed = false;

    const connect = () => {
      socket = new WebSocket(`${WS_URL}/events/${eventId}`);

      socket.onopen = () => setConnected(true);
      socket.onclose = () => {
        setConnected(false);
        if (!closed) retry = setTimeout(connect, 2000); // auto-reconnect
      };
      socket.onmessage = (msg) => {
        try {
          const parsed = wsEventSchema.parse(JSON.parse(msg.data));
          cbRef.current(parsed);
        } catch {
          /* ignore malformed frames */
        }
      };
    };

    connect();
    return () => {
      closed = true;
      if (retry) clearTimeout(retry);
      socket?.close();
    };
  }, [eventId]);

  return { connected };
}
