import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { token } from "../services/api";
export function useLive() {
  const client = useQueryClient();
  useEffect(() => {
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout>;
    async function connect() {
      try {
        const r = await fetch("/api/events", {
          headers: token() ? { Authorization: `Bearer ${token()}` } : {},
          signal: controller.signal,
        });
        if (!r.ok || !r.body) throw new Error("Stream unavailable");
        const reader = r.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";
        while (!controller.signal.aborted) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          let end;
          while ((end = buffer.indexOf("\n\n")) >= 0) {
            const frame = buffer.slice(0, end);
            buffer = buffer.slice(end + 2);
            if (frame.includes("event: scan-")) void client.invalidateQueries();
          }
        }
      } catch {
        /* Visible queries retain their errors; polling is the reconnect fallback. */
      }
      if (!controller.signal.aborted) timer = setTimeout(connect, 5000);
    }
    void connect();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [client]);
}
