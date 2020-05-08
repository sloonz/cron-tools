import React from "react";
import {createRoot} from "react-dom/client";
import {BrowserRouter, Link, Route, Routes} from "react-router";

import type {
  BreadcrumbComponent,
  IJob,
  ILoggedJob,
  IRegisteredJob,
  ISyncResult,
} from "./types";
import Job from "./job";
import JobsList from "./jobs-list";
import Log from "./log";
import "./index.css";

interface ApplicationState {
  loggedJobs?: ILoggedJob[];
  registeredJobs?: IRegisteredJob[];
  syncResult?: ISyncResult;
  isSyncing: boolean;
  showSyncErrors: boolean;
  breadcrumb: BreadcrumbComponent[];
  gen: number;
}

function haveSameBreadcrumb(
  current: BreadcrumbComponent[],
  next: BreadcrumbComponent[],
) {
  return (
    current.length === next.length &&
    current.every(
      (component, index) =>
        component.label === next[index].label &&
        component.to === next[index].to,
    )
  );
}

function SyncModal(props: {syncResult?: ISyncResult; onClose: () => void}) {
  const {syncResult, onClose} = props;

  return (
    <>
      <div
        className="modal"
        tabIndex={-1}
        role="dialog"
        style={{display: "block"}}
      >
        <div className="modal-dialog modal-dialog-scrollable" role="document">
          <div className="modal-content">
            <div className="modal-header">
              <h4 className="modal-title">Sync Error</h4>
              <button
                type="button"
                className="btn-close"
                aria-label="Close"
                onClick={onClose}
              ></button>
            </div>
            <div className="modal-body">
              {syncResult?.error && (
                <>
                  <h5>Error</h5>
                  <p>
                    <span className="font-monospace">{syncResult.error}</span>
                  </p>
                </>
              )}
              {syncResult?.output && (
                <>
                  <h5>Script output</h5>
                  <pre style={{overflow: "visible"}}>
                    <code>{syncResult.output}</code>
                  </pre>
                </>
              )}
            </div>
            <div className="modal-footer">
              <button
                type="button"
                className="btn btn-secondary"
                onClick={onClose}
              >
                Close
              </button>
            </div>
          </div>
        </div>
      </div>
      <div className="modal-backdrop show" />
    </>
  );
}

function ApplicationNav(props: {
  breadcrumb: BreadcrumbComponent[];
  isSyncing: boolean;
  syncResult?: ISyncResult;
  onSync: (event?: React.MouseEvent<HTMLElement>) => void;
  onShowSyncErrors: (event?: React.MouseEvent<HTMLElement>) => void;
}) {
  const {breadcrumb, isSyncing, syncResult, onSync, onShowSyncErrors} = props;

  return (
    <header className="navbar navbar-dark bg-dark">
      <nav className="container-fluid">
        <span className="navbar-text">
          <div className="breadcrumb mb-0 text-light">
            <Link
              className="breadcrumb-item text-light text-decoration-none"
              to="/"
            >
              <i className="bi-house" />
            </Link>
            {breadcrumb.map((component) =>
              component.to ? (
                <Link
                  key={component.label}
                  className="breadcrumb-item text-light text-decoration-none"
                  to={component.to}
                >
                  {component.label}
                </Link>
              ) : (
                <span
                  key={component.label}
                  className="breadcrumb-item text-light"
                >
                  {component.label}
                </span>
              ),
            )}
          </div>
        </span>
        <span className="navbar-text ms-auto">
          Last Refresh:{" "}
          {syncResult?.syncDate
            ? new Date(syncResult.syncDate).toLocaleString()
            : "never"}
        </span>

        {/* minWidth and minHeight are here to avoid navbar resizing when switching between spinner/button */}
        <ul
          className="navbar-nav flex-row align-items-center"
          style={{minHeight: 42}}
        >
          <li className="nav-item" style={{minWidth: 36}}>
            {isSyncing ? (
              <button
                type="button"
                className="nav-link ms-2 py-0 border-0 bg-transparent"
                disabled
                aria-label="Sync in progress"
              >
                <div
                  className="text-light spinner-border spinner-border-sm"
                  style={{height: 24, width: 24}}
                ></div>
              </button>
            ) : (
              <button
                type="button"
                className="nav-link ms-2 py-0 border-0 bg-transparent"
                aria-label="Refresh jobs"
                onClick={onSync}
              >
                <i className="bi-arrow-clockwise text-success fs-3"></i>
              </button>
            )}
          </li>
        </ul>
        <ul className="navbar-nav flex-row align-items-center">
          {syncResult?.success === false && (
            <li className="nav-item">
              <button
                type="button"
                className="nav-link ms-2 py-0"
                aria-label="Show sync errors"
                onClick={onShowSyncErrors}
              >
                <i className="bi-exclamation-triangle text-warning fs-3"></i>
              </button>
            </li>
          )}
        </ul>
      </nav>
    </header>
  );
}

async function loadApplicationData() {
  const syncResult = await fetch("/api/sync", {method: "POST"})
    .then((response) => response.json())
    .catch((error) => {
      console.error(error);
      alert("Failed to do synchronization");
    });

  const [loggedJobs, registeredJobs] = await Promise.all([
    fetch("/api/logs")
      .then((response) => response.json())
      .catch((error) => {
        console.error(error);
        alert("Failed to load logs summary");
      }),
    fetch("/api/jobs")
      .then((response) => response.json())
      .catch((error) => {
        console.error(error);
        alert("Failed to load jobs");
      }),
  ]);
  return {syncResult, loggedJobs, registeredJobs};
}

function Application() {
  const [state, setState] = React.useState<ApplicationState>({
    isSyncing: true,
    showSyncErrors: false,
    breadcrumb: [],
    gen: 0,
  });

  const synchronize = React.useCallback(() => {
    return loadApplicationData().then(
      ({syncResult, loggedJobs, registeredJobs}) => {
        setState((current) => ({
          ...current,
          isSyncing: false,
          syncResult,
          loggedJobs,
          registeredJobs,
          showSyncErrors: true,
          gen: current.gen + 1,
        }));
      },
    );
  }, []);

  React.useEffect(() => {
    void synchronize();
  }, [synchronize]);

  const doSync = React.useCallback(
    (event?: React.MouseEvent<HTMLElement>) => {
      event?.preventDefault();
      setState((current) => ({...current, isSyncing: true}));
      void synchronize();
    },
    [synchronize],
  );

  const doShowSyncErrors = React.useCallback(
    (event?: React.MouseEvent<HTMLElement>) => {
      event?.preventDefault();
      setState((current) => ({...current, showSyncErrors: true}));
    },
    [],
  );

  const closeSyncErrors = React.useCallback(() => {
    setState((current) => ({...current, showSyncErrors: false}));
  }, []);

  const setBreadcrumb = React.useCallback(
    (breadcrumb: BreadcrumbComponent[]) => {
      setState((current) =>
        haveSameBreadcrumb(current.breadcrumb, breadcrumb)
          ? current
          : {...current, breadcrumb},
      );
    },
    [],
  );

  const jobs = React.useMemo<Record<string, IJob>>(() => {
    const nextJobs: Record<string, IJob> = {};

    for (const loggedJob of state.loggedJobs ?? []) {
      nextJobs[loggedJob.jobId] ??= {id: loggedJob.jobId};
      nextJobs[loggedJob.jobId].logs = loggedJob;
    }

    for (const registeredJob of state.registeredJobs ?? []) {
      nextJobs[registeredJob.description.id] ??= {
        id: registeredJob.description.id,
      };
      nextJobs[registeredJob.description.id].info = registeredJob;
    }

    return nextJobs;
  }, [state.loggedJobs, state.registeredJobs]);

  return (
    <div style={{width: "100vw", height: "100vh", overflow: "auto"}}>
      <div className="container">
        <BrowserRouter>
          {state.showSyncErrors && state.syncResult?.success === false && (
            <SyncModal
              syncResult={state.syncResult}
              onClose={closeSyncErrors}
            />
          )}
          <ApplicationNav
            breadcrumb={state.breadcrumb}
            isSyncing={state.isSyncing}
            syncResult={state.syncResult}
            onSync={doSync}
            onShowSyncErrors={doShowSyncErrors}
          />
          <Routes>
            <Route
              path="/"
              element={
                <JobsList
                  key={state.gen}
                  jobs={jobs}
                  setBreadcrumb={setBreadcrumb}
                />
              }
            />
            <Route
              path="/jobs/:jobId"
              element={
                <Job
                  key={state.gen}
                  jobs={jobs}
                  setBreadcrumb={setBreadcrumb}
                />
              }
            />
            <Route
              path="/jobs/:jobId/logs/:logId"
              element={
                <Log
                  key={state.gen}
                  jobs={jobs}
                  setBreadcrumb={setBreadcrumb}
                />
              }
            />
          </Routes>
        </BrowserRouter>
      </div>
    </div>
  );
}

window.addEventListener("DOMContentLoaded", () => {
  const rootEl = document.getElementById("root");
  if (!rootEl) {
    throw new Error("Missing root element");
  }

  createRoot(rootEl).render(<Application />);
});
