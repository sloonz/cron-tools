import React from "react";
import {useParams} from "react-router";

import type {IJob, ILogMetadata, SetBreadcrumb} from "./types";
import {parseLog, formatCommand, formatDuration} from "./common";

export default function Log(props: {
  jobs: Record<string, IJob>;
  setBreadcrumb: SetBreadcrumb;
}) {
  const {setBreadcrumb} = props;
  const {jobId, logId} = useParams<"jobId" | "logId">();

  const [loadedLog, setLoadedLog] = React.useState<{
    jobId: string;
    logId: string;
    data: string;
  }>();
  const [loadedJobLogs, setLoadedJobLogs] = React.useState<{
    jobId: string;
    logs: ILogMetadata[];
  }>();
  const currentLog =
    loadedLog?.jobId === jobId && loadedLog?.logId === logId
      ? loadedLog
      : undefined;
  const logData = currentLog?.data;
  const jobLogs =
    loadedJobLogs?.jobId === jobId ? loadedJobLogs?.logs : undefined;

  React.useEffect(() => {
    if (!jobId || !logId) {
      return;
    }

    fetch(`/api/logs/${encodeURIComponent(jobId)}`)
      .then((res) => res.json())
      .then((logs) => setLoadedJobLogs({jobId, logs}))
      .catch(console.error);
    fetch(`/api/logs/${encodeURIComponent(jobId)}/${encodeURIComponent(logId)}`)
      .then((res) => res.text())
      .then((data) => setLoadedLog({jobId, logId, data}))
      .catch(console.error);
  }, [jobId, logId]);

  React.useEffect(() => {
    if (!jobId || !logId) {
      return;
    }

    const startDate = jobLogs?.find((l) => l.logId === logId)?.startTime;
    const label = startDate
      ? new Date(startDate).toLocaleString()
      : logId!.split(".")[0];
    setBreadcrumb([
      {label: jobId!, to: `/jobs/${encodeURIComponent(jobId!)}`},
      {label},
    ]);
  }, [jobId, logId, jobLogs, setBreadcrumb]);

  const parsedLog = React.useMemo(
    () => (logData !== undefined ? parseLog(logData) : logData),
    [logData],
  );
  const rawBody = React.useMemo(() => {
    if (logData === undefined) {
      return [];
    }
    const lines = logData.replace(/\n$/, "").split("\n");
    return parsedLog?.logType === "ERR" ? lines.slice(1) : lines;
  }, [logData, parsedLog]);

  if (logData === undefined) {
    return (
      <div className="d-flex justify-content-center">
        <div
          className="spinner-border spinner-border-sm mt-5"
          style={{width: "10em", height: "10em"}}
        />
      </div>
    );
  }

  const {logType, startTime, endTime, headers, footers, body} = parsedLog!;

  return (
    <div className="d-flex align-items-center flex-column">
      {logType === "LOG" ? (
        <>
          <div className="container border rounded mt-5">
            <div className="row">
              <div className="col-2 fw-bold">Job ID</div>
              <div className="col-4">{jobId}</div>
              <div className="col-2 fw-bold">Duration</div>
              <div className="col-4">
                {formatDuration(+new Date(endTime!) - +new Date(startTime!))}
              </div>
            </div>
            <div className="row">
              <div className="col-2 fw-bold">Start Date</div>
              <div className="col-4">
                {new Date(startTime!).toLocaleString()}
              </div>
              <div className="col-2 fw-bold">End Date</div>
              <div className="col-4">{new Date(endTime!).toLocaleString()}</div>
            </div>
            <div className="row">
              <div className="col-2 fw-bold">User</div>
              <div className="col-4">{headers!["User"]}</div>
              <div className="col-2 fw-bold">Host</div>
              <div className="col-4">{headers!["Host"]}</div>
            </div>
            <div className="row">
              <div className="col-2 fw-bold">Command</div>
              <div className="col-10 font-monospace">
                {formatCommand(JSON.parse(headers!["Command"]))}
              </div>
            </div>
            <div className="row">
              <div className="col-2 fw-bold">Result</div>
              <div className="col-10 font-monospace">
                {footers!["Exit"] === "code 0" ? (
                  <i className="bi-check-circle text-success" />
                ) : (
                  <i className="bi-x-circle text-danger" />
                )}{" "}
                {footers!["Exit"]}
              </div>
            </div>
          </div>
          {body.length == 0 && (
            <div className="mt-3 border rounded p-2 w-100">No output</div>
          )}
          {body.length > 0 && (
            <div className="mt-3 log-listing border rounded">
              <table>
                {body.map((line, i) => (
                  <tr
                    key={i}
                    className={`bg-${line.stream === "stdout" ? "light" : "warning"}`}
                  >
                    <td className="text-muted font-monospace">
                      {formatDuration(line.offsetMs)}
                    </td>
                    <td>
                      <pre>
                        {line.data}
                        {line.partial ? "" : "\n"}
                      </pre>
                    </td>
                  </tr>
                ))}
              </table>
            </div>
          )}
        </>
      ) : (
        <>
          <div className="w-100 border rounded mt-5 bg-warning p-2">
            {logType == "INVALID"
              ? "Invalid log file, content will be displayed raw below."
              : "Error while generating the log file, output will be displayed raw below."}
          </div>
          {body.length == 0 && (
            <div className="mt-3 border rounded p-2 w-100">No output</div>
          )}
          {rawBody.length > 0 && (
            <pre className="mt-3 w-100 border rounded p-2">
              <code>{rawBody.join("\n")}</code>
            </pre>
          )}
        </>
      )}
    </div>
  );
}
