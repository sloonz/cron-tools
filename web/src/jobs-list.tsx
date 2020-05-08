import React from "react";
import {useNavigate} from "react-router";

import type {IJob, SetBreadcrumb} from "./types";

enum Status {
  ok,
  fail,
  warn,
  missing,
}

export default function JobsList(props: {
  jobs: Record<string, IJob>;
  setBreadcrumb: SetBreadcrumb;
}) {
  const {jobs, setBreadcrumb} = props;

  React.useEffect(() => {
    setBreadcrumb([]);
  }, [setBreadcrumb]);

  const navigate = useNavigate();
  const sortedJobs = React.useMemo(
    () =>
      Object.values(jobs).sort(
        (a, b) =>
          +new Date(b.logs?.lastLog?.time ?? 0) -
          +new Date(a.logs?.lastLog?.time ?? 0),
      ),
    [jobs],
  );
  const jobStatus = (job: IJob) => {
    if (job.logs?.lastLog) {
      if (job.logs?.lastLog?.footers?.["Exit"] === "code 0") {
        if ((job.logs?.failedCount ?? 0) === 0) {
          return Status.ok;
        } else {
          return Status.warn;
        }
      } else {
        return Status.fail;
      }
    } else {
      return Status.missing;
    }
  };
  const classNames = {
    [Status.ok]: "success",
    [Status.fail]: "danger",
    [Status.warn]: "warning",
    [Status.missing]: "warning",
  };
  const icons = {
    [Status.ok]: "check-circle",
    [Status.fail]: "x-circle",
    [Status.warn]: "exclamation-triangle",
    [Status.missing]: "exclamation-triangle",
  };
  return (
    <>
      <div className="d-flex justify-content-center">
        <table className="table table-hover mt-5 w-auto">
          <thead className="table-dark">
            <tr>
              <th>Status</th>
              <th>Job</th>
              <th>Last Run</th>
              <th># Runs</th>
              <th># Successes</th>
              <th># Failures</th>
            </tr>
          </thead>
          <tbody>
            {sortedJobs.map((job) => (
              <tr
                key={job.id}
                className={`table-${classNames[jobStatus(job)]}`}
                style={{cursor: "pointer"}}
                onClick={() => navigate(`/jobs/${encodeURIComponent(job.id)}`)}
              >
                <td className="text-center">
                  <i className={`bi-${icons[jobStatus(job)]}`}></i>
                </td>
                <td>{job.id}</td>
                <td>
                  {job.logs?.lastLog?.time
                    ? new Date(job.logs.lastLog.time).toLocaleString()
                    : "never"}
                </td>
                <td>
                  {(job.logs?.failedCount ?? 0) + (job.logs?.successCount ?? 0)}
                </td>
                <td>{job.logs?.successCount ?? 0}</td>
                <td>{job.logs?.failedCount ?? 0}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
