import React from "react";
import {useParams, useNavigate} from "react-router";

import type {ILogMetadata, IJob, SetBreadcrumb} from "./types";
import {formatDuration, formatCommand} from "./common";

export default function Job(props: {
  jobs: Record<string, IJob>;
  setBreadcrumb: SetBreadcrumb;
}) {
  const {jobs, setBreadcrumb} = props;
  const {jobId} = useParams<"jobId">();
  const navigate = useNavigate();

  const [loadedLogs, setLoadedLogs] = React.useState<{
    jobId: string;
    logs: ILogMetadata[];
  }>();
  const logsList = loadedLogs?.jobId === jobId ? loadedLogs?.logs : undefined;

  React.useEffect(() => {
    if (!jobId) {
      return;
    }

    fetch(`/api/logs/${encodeURIComponent(jobId)}`)
      .then((res) => res.json())
      .then((logs) => setLoadedLogs({jobId, logs}))
      .catch(console.error);
    setBreadcrumb([{label: jobId}]);
  }, [jobId, setBreadcrumb]);

  const sortedLogs = React.useMemo(
    () =>
      [...(logsList ?? [])].sort(
        (a, b) => +new Date(b.startTime) - +new Date(a.startTime),
      ),
    [logsList],
  );

  if (logsList === undefined) {
    return (
      <div className="d-flex justify-content-center">
        <div
          className="spinner-border spinner-border-sm mt-5"
          style={{width: "10em", height: "10em"}}
        />
      </div>
    );
  }

  const jobDesc = jobId ? jobs[jobId]?.info?.description : undefined;

  return (
    <>
      <div className="d-flex align-items-center flex-column">
        {jobDesc && (
          <div className="container border rounded mt-5">
            <div className="row">
              <div className="col-2 fw-bold">Job ID</div>
              <div className="col-4">{jobDesc.id}</div>
              <div className="col-2 fw-bold">Frequency</div>
              <div className="col-4">{jobDesc.frequency}</div>
            </div>
            <div className="row">
              <div className="col-2 fw-bold">User</div>
              <div className="col-4">{jobDesc.user}</div>
              <div className="col-2 fw-bold">Host</div>
              <div className="col-4">{jobDesc.host}</div>
            </div>
            <div className="row">
              <div className="col-2 fw-bold">Command</div>
              <div className="col-10 font-monospace">
                {formatCommand(jobDesc.command)}
              </div>
            </div>
          </div>
        )}
        <table className="table table-hover mt-5 w-auto">
          <thead className="table-dark">
            <tr>
              <th>Status</th>
              <th>Date</th>
              <th>Duration</th>
            </tr>
          </thead>
          <tbody>
            {sortedLogs.map((log) => (
              <tr
                key={log.logId}
                style={{cursor: "pointer"}}
                className={`table-${log.footers["Exit"] === "code 0" ? "success" : "danger"}`}
                onClick={() =>
                  navigate(
                    `/jobs/${encodeURIComponent(jobId!)}/logs/${encodeURIComponent(log.logId)}`,
                  )
                }
              >
                <td className="text-center">
                  {log.footers["Exit"] === "code 0" ? (
                    <i className="bi-check-circle" />
                  ) : (
                    <i className="bi-x-circle" />
                  )}
                </td>
                <td>{new Date(log.startTime).toLocaleString()}</td>
                <td>
                  {formatDuration(
                    +new Date(log.endTime) - +new Date(log.startTime),
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
