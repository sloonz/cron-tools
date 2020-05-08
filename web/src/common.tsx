export function formatDuration(d: number) {
  const totalSecs = Math.round(d / 1000);
  if (totalSecs == 0) {
    return "<1s";
  }

  const hours = Math.floor(totalSecs / 3600);
  const minutes = Math.floor((totalSecs % 3600) / 60);
  const seconds = totalSecs % 60;
  return [
    hours ? `${hours}h` : "",
    hours || minutes ? `${minutes}m` : "",
    `${seconds}s`,
  ].join("");
}

export function formatCommand(command: string[]) {
  function formatArg(arg: string) {
    if (arg.length == 0) {
      return "''";
    } else if (arg.match(/[^\w%+,./:=@_-]/)) {
      return `'${arg.replace("'", `'"'"'`)}'`;
    } else {
      return arg;
    }
  }
  return command.map(formatArg).join(" ");
}

export interface ParsedLog {
  logType: "ERR" | "LOG" | "INVALID";
  body: {
    ts: string;
    offsetMs: number;
    stream: "stdout" | "stderr";
    partial: boolean;
    data: string;
  }[];
  startTime?: string;
  endTime?: string;
  headers?: Record<"Command" | "User" | "Host" | "Job-ID", string>;
  footers?: Record<"Exit", string>;
}

export function parseLog(log: string): ParsedLog {
  const lines = log.split(/\n/);
  if (lines[0] == "CRONERR") {
    return {logType: "ERR", body: []};
  }

  const headers: Record<string, string> = {};
  const footers: Record<string, string> = {};
  const body: ParsedLog["body"] = [];

  let startDateMs: number | null = null;
  let startTime: string | undefined;
  let endTime: string | undefined;

  for (let i = 0; i < lines.length; i++) {
    if (lines[i] === "" && i == lines.length - 1) {
      continue;
    }
    const m = lines[i].match(
      /^(\S+)\s+(stdout|stderr|meta)\s+([FP])(?:\s([\s\S]*))?$/,
    );
    if (!m) {
      console.error("bad CRI line", lines[i]);
      return {logType: "INVALID", body: []};
    }

    const ts = m[1];
    const dateMs = +new Date(ts);
    if (Number.isNaN(dateMs)) {
      console.error("bad timestamp", lines[i]);
      return {logType: "INVALID", body: []};
    }

    const stream = m[2] as "stdout" | "stderr" | "meta";
    const partial = m[3] === "P";

    if (stream === "meta") {
      if (i === 0 && !partial && (m[4] ?? "") === "cronlog-v0") {
        startTime = ts;
        startDateMs = dateMs;
        continue;
      }
      const metaMatch = (m[4] ?? "").match(/^([^:]+):\s*(.*)$/);
      if (!metaMatch) {
        console.error("bad meta line", m[4]);
        return {logType: "INVALID", body: []};
      }

      const key = metaMatch[1];
      const value = metaMatch[2];
      if (key === "Exit") {
        footers[key] = value;
        endTime = ts;
      } else {
        headers[key] = value;
      }
      continue;
    }

    body.push({
      ts,
      offsetMs: startDateMs === null ? 0 : dateMs - startDateMs,
      stream: stream as "stdout" | "stderr",
      partial,
      data: m[4] ?? "",
    });
  }

  return {logType: "LOG", startTime, endTime, headers, footers, body};
}
